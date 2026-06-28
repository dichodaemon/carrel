package registry

import (
	"sort"
	"sync"

	"github.com/google/uuid"
)

// MemRegistry is an in-memory Registry implementation for testing.
type MemRegistry struct {
	mu         sync.RWMutex
	consumers  map[uuid.UUID]Consumer
	aliases    map[string]uuid.UUID // alias → consumer/source ID
	sources    map[uuid.UUID]Source
	entries    map[uuid.UUID]Entry
	deployments map[uuid.UUID]Deployment
	depEntries  map[uuid.UUID][]DeploymentEntry
	consumerSources map[uuid.UUID]map[uuid.UUID]bool // consumerID → set of sourceID
	slots      map[uuid.UUID]Slot
	entrySlots map[uuid.UUID]map[uuid.UUID]int // slotID → entryID → priority
}

func NewMemRegistry() *MemRegistry {
	return &MemRegistry{
		consumers:   make(map[uuid.UUID]Consumer),
		aliases:     make(map[string]uuid.UUID),
		sources:     make(map[uuid.UUID]Source),
		entries:     make(map[uuid.UUID]Entry),
		deployments: make(map[uuid.UUID]Deployment),
		depEntries:  make(map[uuid.UUID][]DeploymentEntry),
		consumerSources: make(map[uuid.UUID]map[uuid.UUID]bool),
		slots:           make(map[uuid.UUID]Slot),
		entrySlots:      make(map[uuid.UUID]map[uuid.UUID]int),
	}
}

func (m *MemRegistry) RegisterConsumer(c Consumer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existingID, ok := m.aliases[c.Alias]; ok && existingID != c.ID {
		return ErrDuplicate
	}
	m.consumers[c.ID] = c
	m.aliases[c.Alias] = c.ID
	return nil
}

func (m *MemRegistry) ResolveConsumer(aliasOrPath string) (Consumer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if id, ok := m.aliases[aliasOrPath]; ok {
		if c, ok := m.consumers[id]; ok {
			return c, nil
		}
	}
	for _, c := range m.consumers {
		if c.Path == aliasOrPath {
			return c, nil
		}
	}
	return Consumer{}, ErrNotFound
}

func (m *MemRegistry) ListConsumers() ([]Consumer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Consumer, 0, len(m.consumers))
	for _, c := range m.consumers {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Alias < out[j].Alias })
	return out, nil
}

func (m *MemRegistry) RegisterSource(s Source) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existingID, ok := m.aliases[s.Alias]; ok && existingID != s.ID {
		return ErrDuplicate
	}
	m.sources[s.ID] = s
	m.aliases[s.Alias] = s.ID
	return nil
}

func (m *MemRegistry) LinkConsumerSource(consumerID, sourceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.consumerSources[consumerID] == nil {
		m.consumerSources[consumerID] = make(map[uuid.UUID]bool)
	}
	m.consumerSources[consumerID][sourceID] = true
	return nil
}

func (m *MemRegistry) ResolveSources(consumerID uuid.UUID) ([]Source, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var universal, specific []Source
	for _, s := range m.sources {
		if s.Scope == ScopeUniversal {
			universal = append(universal, s)
		} else if s.Scope == ScopeTargetSpecific {
			if m.consumerSources[consumerID] != nil && m.consumerSources[consumerID][s.ID] {
				specific = append(specific, s)
			}
		}
	}
	out := append(universal, specific...)
	return out, nil
}

func (m *MemRegistry) ListSources() ([]Source, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Source, 0, len(m.sources))
	for _, s := range m.sources {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		return out[i].Alias < out[j].Alias
	})
	return out, nil
}

func (m *MemRegistry) RegisterEntry(e Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[e.ID] = e
	return nil
}

func (m *MemRegistry) ResolveEntries(sourceIDs []uuid.UUID) ([]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	idSet := make(map[uuid.UUID]bool)
	for _, id := range sourceIDs {
		idSet[id] = true
	}
	var out []Entry
	for _, e := range m.entries {
		if idSet[e.SourceID] {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (m *MemRegistry) RemoveEntry(entryID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[entryID]; !ok {
		return ErrNotFound
	}
	delete(m.entries, entryID)
	return nil
}

func (m *MemRegistry) UpdateEntryMeta(entryID uuid.UUID, updates MetaUpdates) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[entryID]
	if !ok {
		return ErrNotFound
	}
	if updates.ComposeMode != nil {
		e.ComposeMode = **updates.ComposeMode
	}
	m.entries[entryID] = e
	return nil
}

func (m *MemRegistry) RecordDeployment(d Deployment, entries []DeploymentEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deployments[d.ID] = d
	m.depEntries[d.ID] = entries
	return nil
}

func (m *MemRegistry) LastDeployment(consumerID uuid.UUID) (Deployment, []DeploymentEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest *Deployment
	var latestID uuid.UUID
	for id, d := range m.deployments {
		if d.ConsumerID == consumerID && d.SucceededAt != nil {
			if latest == nil || d.AttemptedAt.After(latest.AttemptedAt) {
				copy := d
				latest = &copy
				latestID = id
			}
		}
	}
	if latest == nil {
		return Deployment{}, nil, ErrNoDeployment
	}
	return *latest, m.depEntries[latestID], nil
}

// Slot operations.

func (m *MemRegistry) RegisterSlot(s Slot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slots[s.ID] = s
	return nil
}

func (m *MemRegistry) UpdateSlot(slotID uuid.UUID, updates SlotUpdates) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[slotID]
	if !ok {
		return ErrNotFound
	}
	if updates.Name != nil {
		s.Name = *updates.Name
	}
	if updates.DestPath != nil {
		s.DestPath = *updates.DestPath
	}
	if updates.ComposeMode != nil {
		s.ComposeMode = *updates.ComposeMode
	}
	m.slots[slotID] = s
	return nil
}

func (m *MemRegistry) RemoveSlot(slotID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.slots[slotID]; !ok {
		return ErrNotFound
	}
	delete(m.slots, slotID)
	delete(m.entrySlots, slotID)
	return nil
}

func (m *MemRegistry) ResolveSlots(consumerID uuid.UUID) ([]Slot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Slot
	for _, s := range m.slots {
		if s.ConsumerID == consumerID {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemRegistry) LinkEntrySlot(entryID, slotID uuid.UUID, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entrySlots[slotID] == nil {
		m.entrySlots[slotID] = make(map[uuid.UUID]int)
	}
	m.entrySlots[slotID][entryID] = priority
	return nil
}

func (m *MemRegistry) UnlinkEntrySlot(entryID, slotID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entrySlots[slotID] == nil {
		return ErrNotFound
	}
	if _, ok := m.entrySlots[slotID][entryID]; !ok {
		return ErrNotFound
	}
	delete(m.entrySlots[slotID], entryID)
	return nil
}

// UnlinkAllEntrySlots implements Registry.
func (m *MemRegistry) UnlinkAllEntrySlots(entryID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, slotMap := range m.entrySlots {
		delete(slotMap, entryID)
	}
	return nil
}

func (m *MemRegistry) ResolveEntrySlots(slotID uuid.UUID) ([]SlotEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []SlotEntry
	for eID, priority := range m.entrySlots[slotID] {
		if e, ok := m.entries[eID]; ok {
			out = append(out, SlotEntry{Entry: e, Priority: priority})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Close implements Registry. No-op for in-memory registry.
func (m *MemRegistry) Close() error {
	return nil
}

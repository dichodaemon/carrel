"""Jira CRUD operations."""
from __future__ import annotations

from typing import Any, Optional

from atl import ApiError, InputError
from atl.client import AtlassianClient
from atl.content import adf_to_text, html_to_markdown, text_to_adf


_ACCEPTANCE_CRITERIA = "Acceptance Criteria"


def _custom_field_id(client: AtlassianClient, name: str) -> str:
    """Resolve a field's ID (e.g. ``customfield_10443``) from its display name."""
    resp = client.get("/rest/api/3/field")
    for field in resp.json():
        if field.get("name", "").lower() == name.lower():
            return field["id"]
    raise InputError(f"No Jira field named '{name}' on this instance")


def get_issue(client: AtlassianClient, key: str) -> dict[str, Any]:
    """Fetch a Jira issue and return a flat summary dict."""
    resp = client.get(f"/rest/api/3/issue/{key}?expand=renderedFields,names")
    data = resp.json()
    fields = data.get("fields", {})
    rendered = data.get("renderedFields", {})

    def rich_text(field_id: Optional[str]) -> str:
        # Prefer rendered HTML -> markdown, fall back to ADF -> text.
        if field_id is None:
            return ""
        html = rendered.get(field_id)
        if html:
            return html_to_markdown(html)
        return adf_to_text(fields.get(field_id))

    ac_field = next(
        (fid for fid, fname in data.get("names", {}).items()
         if fname.lower() == _ACCEPTANCE_CRITERIA.lower()),
        None,
    )

    assignee = fields.get("assignee")
    priority = fields.get("priority")
    status = fields.get("status")
    parent = fields.get("parent")

    return {
        "key": data["key"],
        "summary": fields.get("summary", ""),
        "status": status["name"] if status else None,
        "assignee": assignee["displayName"] if assignee else None,
        "assignee_account_id": assignee["accountId"] if assignee else None,
        "priority": priority["name"] if priority else None,
        "labels": fields.get("labels", []),
        "components": [c["name"] for c in fields.get("components") or []],
        "due_date": fields.get("duedate"),
        "parent": parent["key"] if parent else None,
        "description": rich_text("description"),
        "acceptance_criteria": rich_text(ac_field),
        "created": fields.get("created"),
        "updated": fields.get("updated"),
        "url": f"{client.instance}/browse/{data['key']}",
    }


def create_issue(
    client: AtlassianClient,
    project: str,
    issue_type: str,
    summary: str,
    description: Optional[str] = None,
    parent: Optional[str] = None,
    assignee_account_id: Optional[str] = None,
    components: Optional[list[str]] = None,
    due_date: Optional[str] = None,
    acceptance_criteria: Optional[str] = None,
) -> dict[str, Any]:
    """Create a Jira issue and return ``{key, url}``.

    Args:
        parent: Parent issue key (e.g. PROJ-123) to create a child under.
        assignee_account_id: Atlassian account ID to assign the issue to.
        components: Component names (some projects, e.g. FRONTIER, require one).
        due_date: ``YYYY-MM-DD``.
        acceptance_criteria: Text for the "Acceptance Criteria" custom field,
            resolved by name on the instance.
    """
    fields: dict[str, Any] = {
        "project": {"key": project},
        "issuetype": {"name": issue_type},
        "summary": summary,
    }
    if description is not None:
        fields["description"] = text_to_adf(description)
    if parent is not None:
        fields["parent"] = {"key": parent}
    if assignee_account_id is not None:
        fields["assignee"] = {"accountId": assignee_account_id}
    if components:
        fields["components"] = [{"name": name} for name in components]
    if due_date is not None:
        fields["duedate"] = due_date
    if acceptance_criteria is not None:
        ac_field = _custom_field_id(client, _ACCEPTANCE_CRITERIA)
        fields[ac_field] = text_to_adf(acceptance_criteria)

    resp = client.post("/rest/api/3/issue", {"fields": fields})
    result = resp.json()
    key = result["key"]
    return {"key": key, "url": f"{client.instance}/browse/{key}"}


def edit_issue(
    client: AtlassianClient,
    key: str,
    summary: Optional[str] = None,
    description: Optional[str] = None,
) -> None:
    """Update fields on an existing issue."""
    fields: dict[str, Any] = {}
    if summary is not None:
        fields["summary"] = summary
    if description is not None:
        fields["description"] = text_to_adf(description)
    if not fields:
        raise InputError("Nothing to update. Provide --summary or --body.")
    client.put(f"/rest/api/3/issue/{key}", {"fields": fields})


def delete_issue(client: AtlassianClient, key: str) -> None:
    """Delete a Jira issue."""
    client.delete(f"/rest/api/3/issue/{key}")


def transition_issue(
    client: AtlassianClient, key: str, transition_name: str,
) -> None:
    """Transition an issue to a new status by name (case-insensitive substring)."""
    resp = client.get(f"/rest/api/3/issue/{key}/transitions")
    transitions = resp.json().get("transitions", [])

    target = transition_name.lower()
    match = None
    for t in transitions:
        if target in t["name"].lower():
            match = t
            break

    if match is None:
        names = [t["name"] for t in transitions]
        raise InputError(
            f"No transition matching '{transition_name}' for {key}. "
            f"Available: {', '.join(names)}"
        )

    client.post(
        f"/rest/api/3/issue/{key}/transitions",
        {"transition": {"id": match["id"]}},
    )


def search_issues(
    client: AtlassianClient, jql: str,
) -> list[dict[str, Any]]:
    """Run a JQL search and return a list of issue summaries."""
    resp = client.get(
        "/rest/api/3/search/jql",
        params={"jql": jql, "maxResults": "50", "fields": "summary,status,assignee,priority"},
    )
    issues = resp.json().get("issues", [])
    results: list[dict[str, Any]] = []
    for issue in issues:
        fields = issue.get("fields", {})
        assignee = fields.get("assignee")
        status = fields.get("status")
        results.append({
            "key": issue["key"],
            "summary": fields.get("summary", ""),
            "status": status["name"] if status else None,
            "assignee": assignee["displayName"] if assignee else None,
            "url": f"{client.instance}/browse/{issue['key']}",
        })
    return results


def add_comment(
    client: AtlassianClient, key: str, body: str,
) -> dict[str, Any]:
    """Add a comment to an issue and return ``{id}``."""
    resp = client.post(
        f"/rest/api/3/issue/{key}/comment",
        {"body": text_to_adf(body)},
    )
    return {"id": resp.json()["id"]}

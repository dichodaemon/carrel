# Only source zshrc for non-interactive shells — Zsh sources it natively
# for interactive shells (otherwise zshrc runs twice, causing plugin
# double-registration, duplicate PATH entries, and input echo issues).
[[ -o interactive ]] || source ~/.zshrc

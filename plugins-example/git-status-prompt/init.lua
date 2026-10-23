-- Example kacuOS plugin: prints the current git branch when the
-- "shell_prompt" hook fires. Uses kacu.exec_readonly because "git" is on
-- the built-in read-only whitelist — this plugin never needs permissions.exec.

kacu_hooks["shell_prompt"] = function()
    local branch = kacu.exec_readonly("git branch --show-current")
    branch = branch:gsub("%s+$", "") -- trim trailing newline

    if branch ~= "" then
        kacu.print("current branch: " .. branch)
    else
        kacu.print("not a git repository")
    end
end

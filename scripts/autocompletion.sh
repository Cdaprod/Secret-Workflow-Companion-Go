#!/bin/bash

_ghm_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="secret workflow config"

    case "${prev}" in
        secret)
            local secret_opts="add remove list"
            COMPREPLY=( $(compgen -W "${secret_opts}" -- "${cur}") )
            return 0
            ;;
        workflow)
            local workflow_opts="add list"
            COMPREPLY=( $(compgen -W "${workflow_opts}" -- "${cur}") )
            return 0
            ;;
        config)
            local config_opts="store list"
            COMPREPLY=( $(compgen -W "${config_opts}" -- "${cur}") )
            return 0
            ;;
        *)
            COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
            return 0
            ;;
    esac
}

complete -F _ghm_completions ghm
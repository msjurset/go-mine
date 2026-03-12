package main

const zshCompletion = `#compdef go-mine

# Zsh completion script for go-mine
# Install: cp _go-mine ~/.oh-my-zsh/custom/completions/
#   — or — go-mine -completion zsh > /path/to/completions/_go-mine

_arguments -s \
  '(-h -help)'{-h,-help}'[Show help message]' \
  '-version[Print version and exit]' \
  '-completion[Print shell completion script]:shell:(zsh bash)' \
  '-generate[Generate sample data instead of loading a file]' \
  '-rows[Number of rows to generate (with -generate)]:rows:' \
  '-info[Print data summary to stdout and exit]' \
  '*:input file:_files -g "*.{csv,tsv,parquet,json}"'
`

const bashCompletion = `# Bash completion script for go-mine
# Install: go-mine -completion bash > /etc/bash_completion.d/go-mine
#   — or — go-mine -completion bash >> ~/.bashrc

_go_mine() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="-generate -rows -info -version -completion -h -help"

    case "$prev" in
        -completion)
            COMPREPLY=( $(compgen -W "zsh bash" -- "$cur") )
            return 0
            ;;
        -rows)
            return 0
            ;;
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
        return 0
    fi

    COMPREPLY=( $(compgen -f -X '!*.@(csv|tsv|parquet|json)' -- "$cur") )
}

complete -F _go_mine go-mine
`

package main

import (
	"fmt"
	"os"
)

// runCompletion генерирует shell completions для указанного shell.
func runCompletion(shell string) error {
	switch shell {
	case "bash":
		fmt.Print(genBash())
	case "zsh":
		fmt.Print(genZsh())
	case "fish":
		fmt.Print(genFish())
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
	return nil
}

// genBash генерирует bash completions script.
func genBash() string {
	return `# bash completion for logt
_logt() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="--path --level --buffer --max-buffer --theme --forward --since --until --json --headless --tail --stats --export --color --help --version -p -l -b -m -t -f -S -U -j -H -n -s -e -h -v"

    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}
complete -F _logt logt
`
}

// genZsh генерирует zsh completions script.
func genZsh() string {
	return `# zsh completion for logt
#compdef logt

_logt() {
    local -a opts
    opts=(
        '--path[-Paths to files or glob patterns]:path:_files'
        '--level[-Log level filter (debug,info,warn,error)]:level:(debug info warn error)'
        '--buffer[-Buffer size]:size:_numbers'
        '--max-buffer[-Max buffer size]:size:_numbers'
        '--theme[-Theme (dark,light)]:theme:(dark light)'
        '--forward[-Export logs to file or stdout]:path:_files'
        '--since[-Filter from time (1h, 30m, 2024-01-15)]:time:'
        '--until[-Filter until time (1h, 30m, 2024-01-15)]:time:'
        '--json[-JSON Path filter]:filter:'
        '--headless[Run without TUI (CLI mode)]'
        '--tail[-Last N lines (0 = all)]:n:_numbers'
        '--stats[Show aggregated statistics]'
        '--export[-Export bookmarks to file]:path:_files'
        '--color[-Color mode (always, never, auto)]:mode:(always never auto)'
        '--help[Show help]'
        '--version[Show version]'
        '-p[Paths to files or glob patterns]'
        '-l[Log level filter]'
        '-b[Buffer size]'
        '-m[Max buffer size]'
        '-t[Theme]'
        '-f[Export logs]'
        '-S[Filter from time]'
        '-U[Filter until time]'
        '-j[JSON Path filter]'
        '-H[Run without TUI]'
        '-n[Last N lines]'
        '-s[Show statistics]'
        '-e[Export bookmarks]'
        '-h[Show help]'
        '-v[Show version]'
    )

    _arguments -s -S $opts
    _files
}

_logt "$@"
`
}

// genFish генерирует fish completions script.
func genFish() string {
	return `# fish completion for logt
complete -c logt -l path -s p -d 'Paths to files or glob patterns' -r -F
complete -c logt -l level -s l -d 'Log level filter' -r -f -a "debug info warn error"
complete -c logt -l buffer -s b -d 'Buffer size' -r
complete -c logt -l max-buffer -s m -d 'Max buffer size' -r
complete -c logt -l theme -s t -d 'Theme' -r -f -a "dark light"
complete -c logt -l forward -s f -d 'Export logs to file or stdout' -r -F
complete -c logt -l since -s S -d 'Filter from time' -r
complete -c logt -l until -s U -d 'Filter until time' -r
complete -c logt -l json -s j -d 'JSON Path filter' -r
complete -c logt -l headless -s H -d 'Run without TUI'
complete -c logt -l tail -s n -d 'Last N lines' -r
complete -c logt -l stats -s s -d 'Show aggregated statistics'
complete -c logt -l export -s e -d 'Export bookmarks to file' -r -F
complete -c logt -l color -d 'Color mode' -r -f -a "always never auto"
complete -c logt -l help -s h -d 'Show help'
complete -c logt -l version -s v -d 'Show version'
`
}

// completionCmd запускает subcommand для генерации completions.
func completionCmd(args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: logt completion <bash|zsh|fish>")
		os.Exit(1)
	}

	shell := args[2]
	if err := runCompletion(shell); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

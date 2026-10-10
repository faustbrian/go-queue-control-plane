#!/usr/bin/env bash
set -euo pipefail

if ! command -v awk >/dev/null; then
    printf 'required documentation scanner is unavailable: awk\n' >&2
    exit 1
fi

required=(
    README.md
    CHANGELOG.md
    CONTRIBUTING.md
    SECURITY.md
    SUPPORT.md
    LICENSE
    client/desired_state_example_test.go
    docs/api.md
    docs/architecture.md
    docs/cli.md
    docs/compatibility.md
    docs/deployment.md
    docs/faq.md
    docs/hardening.md
    docs/horizon-migration.md
    docs/kubernetes.md
    docs/operations.md
    docs/performance.md
    docs/releasing.md
    docs/security.md
    docs/ui.md
)

markdown=(README.md CHANGELOG.md CONTRIBUTING.md SECURITY.md SUPPORT.md docs/*.md)

for file in "${required[@]}"; do
    if [[ ! -s "${file}" ]]; then
        printf 'required documentation is missing or empty: %s\n' "${file}" >&2
        exit 1
    fi
done

for file in "${markdown[@]}"; do
    if [[ "$(head -n 1 "${file}")" != '# '* ]]; then
        printf 'documentation must start with one title: %s\n' "${file}" >&2
        exit 1
    fi
    fences="$(awk '/^```/ { count++ } END { print count + 0 }' "${file}")"
    if (( fences % 2 != 0 )); then
        printf 'unbalanced fenced code block: %s\n' "${file}" >&2
        exit 1
    fi
done

if awk '/QUEUE_CONTROL_TOKEN|\/Users\// {
    printf "%s:%d\n", FILENAME, FNR
    found = 1
} END { exit !found }' "${markdown[@]}"; then
    printf 'documentation contains a stale credential or local path\n' >&2
    exit 1
else
    scan_status=$?
    if [[ "${scan_status}" -ne 1 ]]; then
        printf 'documentation text scan failed\n' >&2
        exit "${scan_status}"
    fi
fi

perl -MFile::Basename=dirname -MFile::Spec -e '
    while (<>) {
        while (/\[[^]]+\]\(([^)#]+)(?:#[^)]*)?\)/g) {
            my $link = $1;
            next if $link =~ m{^(?:https?://|mailto:)};
            my $target = File::Spec->rel2abs($link, dirname($ARGV));
            die "missing relative link: $ARGV:$link\n" unless -e $target;
        }
    }
' "${markdown[@]}"

go test ./client -run '^ExampleClient_DesiredStateReader$' -count=1

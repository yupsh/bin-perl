#!/bin/sh
# Integration checks for yup-perl, run inside a Debian container with the real
# `perl` installed.
#
# yup-perl is a thin wrapper that forks the system perl: it maps its own
# -n/-p/-a switches to perl's, appends `-e SCRIPT`, and streams stdin through.
# So its output must be byte-identical to invoking perl directly. The reference
# below is therefore real `perl` with the equivalent switches.
#
# parity FLAGS SCRIPT INPUT  — yup-perl FLAGS SCRIPT  must match
#                              perl FLAGS -e SCRIPT, both reading INPUT on stdin.
# assert WANT FLAGS SCRIPT INPUT — yup-perl must produce WANT exactly (used for
#                              behavior with no direct flag-for-flag reference).
set -eu

fails=0

# parity: FLAGS is the shared switch string (e.g. "-p", "-n -a", or "" for none),
# passed verbatim to both yup-perl and perl; perl additionally needs -e before
# the script.
parity() {
	flags=$1
	script=$2
	input=$3
	ours=$(printf '%s' "$input" | yup-perl $flags "$script" 2>/dev/null || true)
	ref=$(printf '%s' "$input" | perl $flags -e "$script" 2>/dev/null || true)
	if [ "$ours" = "$ref" ]; then
		printf 'ok    parity  perl %s %s\n' "$flags" "$script"
	else
		printf 'FAIL  parity  perl %s %s\n        perl: %s\n        ours: %s\n' "$flags" "$script" "$ref" "$ours"
		fails=$((fails + 1))
	fi
}

assert() {
	want=$1
	flags=$2
	script=$3
	input=$4
	got=$(printf '%s' "$input" | yup-perl $flags "$script" 2>/dev/null || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    assert  perl %s %s\n' "$flags" "$script"
	else
		printf 'FAIL  assert  perl %s %s\n        want: %s\n        got:  %s\n' "$flags" "$script" "$want" "$got"
		fails=$((fails + 1))
	fi
}

lines='alpha
beta
gamma
'
fields='one two three
four five six
'

# -p (print each line after running SCRIPT): uppercase and substitution.
parity '-p' '$_=uc' "$lines"
parity '-p' 's/a/A/g' "$lines"
parity '-p' 'chomp; $_="[$_]\n"' "$lines"

# -n (loop without auto-print): the SCRIPT must print explicitly.
parity '-n' 'print uc($_)' "$lines"
parity '-n' 'print if /beta/' "$lines"

# -a autosplit, combined with -p / -n: @F holds the whitespace-split fields.
parity '-p -a' '$_=join(":",@F)."\n"' "$fields"
parity '-n -a' 'print "$F[0]\n"' "$fields"

# No mode switch: SCRIPT runs once (perl's BEGIN/END-free one-shot), stdin unread.
parity '' 'print "constant\n"' "$lines"

# Documented behavior: yup-perl's -p sets perl's -p (which implies -n); the
# wrapper's own -p option also enables its loop flag, but the forked perl only
# ever sees -p, so output equals plain `perl -p`.
assert "$(printf 'ALPHA\nBETA\nGAMMA')" '-p' '$_=uc' "$lines"

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'

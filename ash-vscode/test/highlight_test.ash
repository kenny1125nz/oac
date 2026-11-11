#!mockagent:1.0
#!compact mode=on strategy=truncate window=32000

# ── Comments ──────────────────────────────────────────────────────────
# this is a line comment

# ── Variables & Assignment ─────────────────────────────────────────────
NAME = "ash"
VERSION = "0.1.0"
SCORE = 100
PI = 3.14
DEBUG = true
TEMP = -5

# ── Variable References ────────────────────────────────────────────────
print $NAME
print ${NAME}
print "exit code: $?"
print "stderr: ${stderr}"
print "stdout: ${stdout}"
print "report: ${report}"

# ── Strings with interpolation ────────────────────────────────────────
MSG = "hello ${NAME}, version ${VERSION}"
PRICE = "costs \$5.00"
ESCAPED = "line1\nline2\tindented"

# ── Text Blocks ───────────────────────────────────────────────────────
PROMPT = ```
Fix the login bug in src/auth/login.ts.
The error is "Invalid token" when the JWT expires.
```

# ── Command Substitution ──────────────────────────────────────────────
FILES = $(find src -name '*.ts')
MSG = "branch: $(git branch --show-current)"

# ── File Paths ────────────────────────────────────────────────────────
do @skills/review.md with subagent reviewer
do @'play_step${n}.md' with subagent coder

# ── Control Flow ──────────────────────────────────────────────────────
if $? == 0 {
  print "success"
} else if $SCORE > 0.8 {
  print "good enough"
} else {
  print "fail"
  exit 1
}

for FILE in $FILES {
  exec eslint $FILE
}

while $COUNT < 5 {
  COUNT = $COUNT + 1
}

# ── Functions ─────────────────────────────────────────────────────────
fn greet(NAME) {
  print "hello " + $NAME
}

fn add(A, B) {
  return $A + $B
}

fn factorial(N) {
  if $N <= 1 {
    return 1
  }
  return $N * factorial($N - 1)
}

# ── Function Calls ────────────────────────────────────────────────────
greet("world")
RESULT = add(3, 4)
FACT5 = factorial(5)

# ── Agent Calls ───────────────────────────────────────────────────────
do "Fix login bug" with subagent bug-fixer
do "Review code" with subagent code-reviewer using gpt-4
do $PROMPT with subagent fixer compact "truncate 32000"

# ── Try Blocks ────────────────────────────────────────────────────────
try {
  do "Fix errors" with subagent fixer
} fail {
  do "Retry: ${stderr}" with subagent fixer
} upto 3

try {
  do "Refactor" with subagent refactorer
} evaluate with {
  do @check.md with subagent checker
} accept {
  print "accepted"
} partial {
  do "Refine: ${report}" with subagent refactorer
} fail {
  do "Reset: ${report}" with subagent refactorer
} upto 3

# ── Parallel & Background ─────────────────────────────────────────────
wait {
  do "Review a.ts" with subagent reviewer
  do "Review b.ts" with subagent reviewer
}

exec npm install &
print "continues immediately"
wait

# ── Built-ins ─────────────────────────────────────────────────────────
exec npm test
print "done"
env HOME
include "lib/utils.ash"
exit 0

# ── Operators ─────────────────────────────────────────────────────────
X == 10
Y != 20
A > 5
B < 10
C >= 3
D <= 8
SUM = $A + $B
DIF = $A - $B
PROD = $A * $B
QUOT = $A / $B
OK = $X and $Y
DONE = not $FAILED
CHOICE = $A or $B

#!mockagent:1.0
#!compact mode=on strategy=truncate window=32000

# ============================================================================
# ash language test suite
# Covers all major features of the language
# Run: ash test_all.ash
# ============================================================================

# ---------------------------------------------------------------------------
# 1. Variables and assignment
# ---------------------------------------------------------------------------
NAME = "ash"
VERSION = "0.1.0"
AUTHOR = "test suite"
print NAME + " version " + VERSION

# Integer variable
COUNT = 0
SCORE = 100
print "score: " + SCORE

# Float variable
PI = 3.14
print "pi: " + PI

# Bool variable
DEBUG = true
print "debug mode: " + DEBUG

# ---------------------------------------------------------------------------
# 2. Expressions: arithmetic
# ---------------------------------------------------------------------------
A = 10
B = 20
SUM = A + B
DIV = 7 / 2
NEG = -5
print "10 + 20 = " + SUM
print "7 / 2 = " + DIV
print "-5 = " + NEG

# Compound: 1 + 2 * 3 (multiplication has higher precedence)
RES = 1 + 2 * 3
print "1 + 2 * 3 = " + RES

# Parenthesized: (1 + 2) * 3
RES2 = (1 + 2) * 3
print "(1 + 2) * 3 = " + RES2

# ---------------------------------------------------------------------------
# 3. Expressions: comparison and boolean
# ---------------------------------------------------------------------------
if SUM > 5 {
  print "comparison: sum > 5 (true)"
}

if 5 == 5 {
  print "comparison: 5 == 5 (true)"
}

if 3 != 4 {
  print "comparison: 3 != 4 (true)"
}

if true and true {
  print "boolean: true and true (true)"
}

if true or false {
  print "boolean: true or false (true)"
}

if not false {
  print "boolean: not false (true)"
}

if not $? {
  print "exit: \$? is 0 (truthy)"
}

# Compound expression
X = 10
if (X > 0 and X < 20) {
  print "compound: X in range (true)"
}

# ---------------------------------------------------------------------------
# 4. String interpolation
# ---------------------------------------------------------------------------
# ${NAME} inside strings is replaced with the variable value
print "interpolation: hello ${NAME} version ${VERSION}"

# Escaped dollar sign
print "escaped: price is \$5.00"

# ---------------------------------------------------------------------------
# 5. Control flow: if / else if / else
# ---------------------------------------------------------------------------
VAL = 42
if VAL > 100 {
  print "if-else: val > 100"
} else if VAL > 10 {
  print "if-else: val > 10 (this should print)"
} else {
  print "if-else: val <= 10"
}

# Short if
FLAG = false
if FLAG {
  print "if: FLAG is true (should NOT print)"
}

if not FLAG {
  print "if: FLAG is false (this should print)"
}

# ---------------------------------------------------------------------------
# 6. For loop (iterate over newline-separated string)
# ---------------------------------------------------------------------------
print "for loop items:"
for ITEM in "apple
banana
cherry" {
  print "  - " + ITEM
}

# For loop with break
print "for loop with break:"
for X in "a
b
c
d" {
  if X == "c" {
    break
  }
  print "  " + X
}

# For loop with continue
print "for loop with continue:"
for X in "a
b
c
d" {
  if X == "b" {
    continue
  }
  print "  " + X
}

# ---------------------------------------------------------------------------
# 7. While loop
# ---------------------------------------------------------------------------
print "while loop:"
I = 0
while I < 3 {
  print "  i = " + I
  I = I + 1
}

# ---------------------------------------------------------------------------
# 8. Functions
# ---------------------------------------------------------------------------
fn greet(NAME) {
  print "hello " + NAME
}

fn add(A, B) {
  return A + B
}

fn factorial(N) {
  if N <= 1 {
    return 1
  }
  return N * factorial(N - 1)
}

greet("world")
greet("ash")

RES3 = add(5, 7)
print "add(5, 7) = " + RES3

FACT5 = factorial(5)
print "factorial(5) = " + FACT5

# ---------------------------------------------------------------------------
# 9. Env variable access
# ---------------------------------------------------------------------------
# Note: if ASH_TEST_VAR is not set, this will hit the fail block
try {
  env ASH_TEST_VAR
  print "env: ASH_TEST_VAR = " + ASH_TEST_VAR
} fail {
  print "env: ASH_TEST_VAR not set (skipping)"
} upto 1

# ---------------------------------------------------------------------------
# 10. Command substitution
# ---------------------------------------------------------------------------
# Note: this runs a shell command and captures its output
WHOAMI = $(whoami)
print "command: whoami = " + WHOAMI

DATE = $(date +%Y)
print "command: year = " + DATE

# ---------------------------------------------------------------------------
# 11. Exit code tracking
# ---------------------------------------------------------------------------
print "exit code: \$? = " + $?

# Run a failing command (captured via exec)
# exec false
# print "after false: \$? = " + $?

# ---------------------------------------------------------------------------
# 12. Try blocks (binary)
# ---------------------------------------------------------------------------
print "binary try (success):"
try {
  print "  try body executed"
} upto 1

print "binary try (with fail):"
try {
  print "  try body (this will be shown)"
} fail {
  print "  fail block (should NOT show)"
} upto 1

# ---------------------------------------------------------------------------
# 13. Try evaluate with (accept/fail)
# ---------------------------------------------------------------------------
print "eval try (accepted):"
try {
  X = 1
} evaluate with {
  X == 1
} accept {
  print "  accepted (X was 1)"
} upto 1

print "eval try (rejected):"
try {
  X = 0
} evaluate with {
  X == 1
} accept {
  print "  accepted (should NOT show)"
} fail {
  print "  rejected (X was not 1)"
} upto 1

# ---------------------------------------------------------------------------
# 14. Scope and shadowing
# ---------------------------------------------------------------------------
SC = "global"
print "scope: " + SC

if true {
  SC = "local"
  print "scope inside block: " + SC
}

print "scope after block: " + SC

# ---------------------------------------------------------------------------
# 15. Text blocks (literal string with escapes)
# ---------------------------------------------------------------------------
BLOCK = ```line1
line2
line3```
print "text block:"
print BLOCK

# ---------------------------------------------------------------------------
# 16. Background with & postfix and wait with parallel block
# ---------------------------------------------------------------------------
print "background (& postfix):"
{ print "  bg task 1" } &
print "  main continues while bg runs"
wait
print "  all background tasks done"

print "parallel wait block:"
wait {
  print "  parallel task A"
  print "  parallel task B"
}
print "  both tasks completed"

# ---------------------------------------------------------------------------
# 17. Compact mode
# ---------------------------------------------------------------------------
# compact directives are always string-based per spec
compact "truncate 32000"
print "compact: truncate directive"

compact "summarize"
print "compact: summarize directive"

# ---------------------------------------------------------------------------
# 18. Complex expressions
# ---------------------------------------------------------------------------
# Mixed arithmetic and comparison
SCORE = 85
GRADE = ""

fn getGrade(S) {
  if S >= 90 {
    return "A"
  }
  if S >= 80 {
    return "B"
  }
  if S >= 70 {
    return "C"
  }
  return "F"
}

GRADE = getGrade(SCORE)
print "score " + SCORE + " = grade " + GRADE

# ---------------------------------------------------------------------------
# 19. Nested functions and recursion
# ---------------------------------------------------------------------------
fn fib(N) {
  if N <= 1 {
    return N
  }
  return fib(N - 1) + fib(N - 2)
}

FIB6 = fib(6)
print "fib(6) = " + FIB6

# ---------------------------------------------------------------------------
# 20. Arrays
# ---------------------------------------------------------------------------
print "arrays:"

# Array literal
FRUITS = ["apple", "banana", "cherry"]
print "  fruits = " + FRUITS

# Array length
print "  len(fruits) = " + len(FRUITS)

# Array index access
print "  fruits[0] = " + FRUITS[0]
print "  fruits[1] = " + FRUITS[1]
print "  fruits[2] = " + FRUITS[2]

# Empty array
EMPTY = []
print "  empty array: " + EMPTY
print "  len(empty) = " + len(EMPTY)

# Array concatenation
A = [1, 2, 3]
B = [4, 5]
C = A + B
print "  A + B = " + C

# For loop over array
print "  for over array:"
for ITEM in ["x", "y", "z"] {
  print "    " + ITEM
}

# Array of mixed types
MIXED = ["hello", 42, true]
print "  mixed array: " + MIXED
print "  len(mixed) = " + len(MIXED)

# Nested indexing in expression
NUMS = [10, 20, 30]
IDX = 1
print "  NUMS[IDX] = " + NUMS[IDX]

# ---------------------------------------------------------------------------
# 21. Agent operations (do with subagent)
# ---------------------------------------------------------------------------
print "agent operations:"
do "write hello world" with subagent code-generator
print "  stdout: " + stdout

do "implement fibonacci" with subagent code-reviewer using "gpt-4"
print "  stdout: " + stdout
print "  stderr: " + stderr
print "  exit code: " + $?

do "summarize project" with subagent documenter compact "truncate 16000"
print "  agent called with compact directive"

do @"testdata/prompt.md" with subagent doc
print "  loaded prompt from @ file"

# ---------------------------------------------------------------------------
# 22. Exit (must be last)
# ---------------------------------------------------------------------------
exit 0

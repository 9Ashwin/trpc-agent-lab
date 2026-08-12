#!/usr/bin/env python3
"""Print the first N Fibonacci numbers."""
import sys


def fib(n: int) -> list[int]:
    seq = [0, 1]
    while len(seq) < n:
        seq.append(seq[-1] + seq[-2])
    return seq[:n]


if __name__ == "__main__":
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 10
    print(" ".join(str(x) for x in fib(n)))

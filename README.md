# MCA

It's a math compiler to asm.

Basically, you can pass programming language math as a string, and get back a file in asm that do this math.

The name:

- **M** math
- **C** compiler
- **A** asm

## Examples

```
10 + 10
```

```
2 + 2 / 2
```

```
(2 + 2) / 2
```

```
2 * 2 / (2 + (2 - 3))
```

```
5.5 * 2 / 3
```

## Debugging

you can export `MCA_LOG_ENABLED` as `1` to enable logging.

```bash
export MCA_LOG_ENABLED=1
```

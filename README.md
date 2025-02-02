# MCA

It's a math compiler to asm.

Basically, you can pass programming language math as a string, and get back a file in asm that do this math.

The name:

- **M** math
- **C** compiler
- **A** asm

![error handling](https://github.com/user-attachments/assets/5d7906aa-09e9-4c29-a8b4-b8422a441b7c "error handling")

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

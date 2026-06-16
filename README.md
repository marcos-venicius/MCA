# MCA

<img width="1920" height="1048" alt="image" src="https://github.com/user-attachments/assets/deb8e1b7-e375-4a14-ace1-9f335ad23400" />

It's a math expression calculator.

Basically, you can pass programming language math as a string, and evaluate it (which is very useful).

The name:

- **M** math
- **CA** calculator

![error handling](https://github.com/user-attachments/assets/5d7906aa-09e9-4c29-a8b4-b8422a441b7c "error handling")

## Available operators

- `*` times
- `-` subtract
- `+` sum
- `/` divide
- `%` modules
- `^` power
- `!` factorial

all the numbers will be handled as C Doubles.

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

```
5! - 20
```

```
-100 * -1
```

## We have some bugs yet

I'm not properly parsing the tokens before mount the ast, so expressions like `
` will kinda work, but with the wrong result.
Or for example `2 (^ 2 + 2)`, will return a wrong result without any errors.

## Debugging

you can export `MCA_LOG_ENABLED` as `1` to enable logging.

```bash
export MCA_LOG_ENABLED=1
```

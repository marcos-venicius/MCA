# MCA Language

_It's an advanced **M**ath expression **CA**lculator._

MCA comes with a robust standard library of mathematical functions and constants. Since all values in MCA are `double` floating-point numbers, all functions expect and return `double` numbers. Function calls require parentheses even if they take zero arguments.

### Early Returns (`break`)
Because loops evaluate to values, you can use `break <expression>` to stop the loop and assign its result immediately.
```mca
x = 0;
result = while {
    x = x + 1;
    if x == 10 { 
        break 42;  # The loop evaluating this break will evaluate to 42
    };
};
# result is 42
```

### Trigonometry
*Note: The trigonometric functions evaluate arguments in radians.*
- **`sin(x)`**: Returns the sine of an angle `x` (in radians).
- **`cos(x)`**: Returns the cosine of an angle `x` (in radians).
- **`tan(x)`**: Returns the tangent of an angle `x` (in radians).
- **`rad(x)`**: Converts degrees to radians. Useful for wrapping arguments: `sin(rad(90))`.
- **`deg(x)`**: Converts radians to degrees.

**_...For a detailed view, checkout [syntax.md](./syntax.md) file._**

## Debugging

you can export `MCA_LOG_ENABLED` as `1` to enable logging.

```bash
export MCA_LOG_ENABLED=1
```

## Building the tool

If you want to build the tool you can use:

```bash
make
```

If you want an omptized version:

```
MCA_OPTIMIZE=1 make
```

## Building the test cases

```bash
make bin/test
```

Running the test cases

```bash
./bin/test
```

## Tool usage help

```bash
USAGE: mca [math] [flags]

    -i   [file]         evaluate math inside a file
    -h                  show this help

error: please, provide some math or -i flag
```

## Example

**Single expression example:**

```bash
mca 'max(abs(-12), 8) * sin(rad(30)) + (16 / 2)'
```

The result should be `14`

**Multi expressions example:**

```bash
mca -i ./math.mca
```

> Note that you could separate expressions inline like `mca '1 + 1 ; 2 + 2'`; I used the file here just to make the example easy.
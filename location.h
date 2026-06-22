#ifndef LOCATION_H_
#define LOCATION_H_

typedef struct {
    const char *filename;

    int col, line;
} M_Location;

#endif // LOCATION_H_

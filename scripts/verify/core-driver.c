#define _POSIX_C_SOURCE 200809L
#include <stdio.h>
#include <stdlib.h>
#include "lithe_core.h"

/* One long-lived Core owns the real plugin process. JSON lines are only a
 * test-driver transport; the plugin itself uses the official Go SDK. */
int main(void) {
    char *line = NULL;
    size_t size = 0;
    while (getline(&line, &size, stdin) > 0) {
        char *result = lithe_core_execute_json(line);
        if (result == NULL) { free(line); return 1; }
        puts(result);
        fflush(stdout);
        lithe_core_free_string(result);
    }
    free(line);
    return 0;
}

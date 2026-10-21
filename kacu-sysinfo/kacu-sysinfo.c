/*
 * kacu-sysinfo.c
 *
 * Small, dependency-free utility that reads system stats directly from
 * /proc and prints them either as plain text or JSON.
 *
 * Design goals (this is the whole point of writing it in C instead of
 * shelling out to `free`/`lscpu` from kacuctl):
 *   - zero heap allocation beyond a couple of fixed-size stack buffers
 *   - zero external dependencies (no libjson, no libproc, just libc)
 *   - single static binary, a few KB, sub-millisecond runtime
 *
 * Usage:
 *   kacu-sysinfo            plain text
 *   kacu-sysinfo --json     JSON on stdout, for kacuctl to parse
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define LINE_MAX_LEN 256

typedef struct {
    long mem_total_kb;
    long mem_available_kb;
    long swap_total_kb;
    long swap_free_kb;
} MemInfo;

typedef struct {
    char model_name[128];
    int cpu_count;
} CpuInfo;

typedef struct {
    double load1;
    double load5;
    double load15;
} LoadInfo;

/* Reads a single "Key:    123 kB" style value out of /proc/meminfo.
 * Returns 1 on success, 0 if the key wasn't found. */
static int meminfo_read_field(const char *key, long *out_value) {
    FILE *f = fopen("/proc/meminfo", "r");
    if (!f) {
        return 0;
    }

    char line[LINE_MAX_LEN];
    size_t key_len = strlen(key);
    int found = 0;

    while (fgets(line, sizeof(line), f)) {
        if (strncmp(line, key, key_len) == 0) {
            /* Skip past "Key:" and any whitespace, then parse the number. */
            char *p = line + key_len;
            while (*p == ' ' || *p == '\t') p++;
            *out_value = atol(p);
            found = 1;
            break;
        }
    }

    fclose(f);
    return found;
}

static int read_mem_info(MemInfo *mem) {
    memset(mem, 0, sizeof(MemInfo));
    int ok = 1;
    ok &= meminfo_read_field("MemTotal:", &mem->mem_total_kb);
    ok &= meminfo_read_field("MemAvailable:", &mem->mem_available_kb);
    ok &= meminfo_read_field("SwapTotal:", &mem->swap_total_kb);
    ok &= meminfo_read_field("SwapFree:", &mem->swap_free_kb);
    return ok;
}

static int read_cpu_info(CpuInfo *cpu) {
    memset(cpu, 0, sizeof(CpuInfo));
    strcpy(cpu->model_name, "unknown");

    FILE *f = fopen("/proc/cpuinfo", "r");
    if (!f) {
        return 0;
    }

    char line[LINE_MAX_LEN];
    int got_model = 0;

    while (fgets(line, sizeof(line), f)) {
        if (!got_model && strncmp(line, "model name", 10) == 0) {
            char *colon = strchr(line, ':');
            if (colon) {
                colon += 2; /* skip ": " */
                size_t len = strlen(colon);
                if (len > 0 && colon[len - 1] == '\n') {
                    colon[len - 1] = '\0';
                }
                strncpy(cpu->model_name, colon, sizeof(cpu->model_name) - 1);
                got_model = 1;
            }
        }
        if (strncmp(line, "processor", 9) == 0) {
            cpu->cpu_count++;
        }
    }

    fclose(f);
    return 1;
}

static int read_load_info(LoadInfo *load) {
    FILE *f = fopen("/proc/loadavg", "r");
    if (!f) {
        load->load1 = load->load5 = load->load15 = 0.0;
        return 0;
    }
    int matched = fscanf(f, "%lf %lf %lf", &load->load1, &load->load5, &load->load15);
    fclose(f);
    return matched == 3;
}

static void print_plain(const MemInfo *mem, const CpuInfo *cpu, const LoadInfo *load) {
    printf("kacuOS system info\n");
    printf("------------------\n");
    printf("CPU model:    %s\n", cpu->model_name);
    printf("CPU cores:    %d\n", cpu->cpu_count);
    printf("Memory total: %.1f MB\n", mem->mem_total_kb / 1024.0);
    printf("Memory used:  %.1f MB\n",
           (mem->mem_total_kb - mem->mem_available_kb) / 1024.0);
    printf("Memory avail: %.1f MB\n", mem->mem_available_kb / 1024.0);
    if (mem->swap_total_kb > 0) {
        printf("Swap total:   %.1f MB\n", mem->swap_total_kb / 1024.0);
        printf("Swap free:    %.1f MB\n", mem->swap_free_kb / 1024.0);
    } else {
        printf("Swap:         none (or zram not yet active)\n");
    }
    printf("Load avg:     %.2f %.2f %.2f\n", load->load1, load->load5, load->load15);
}

static void print_json(const MemInfo *mem, const CpuInfo *cpu, const LoadInfo *load) {
    printf("{\n");
    printf("  \"cpu_model\": \"%s\",\n", cpu->model_name);
    printf("  \"cpu_cores\": %d,\n", cpu->cpu_count);
    printf("  \"mem_total_kb\": %ld,\n", mem->mem_total_kb);
    printf("  \"mem_available_kb\": %ld,\n", mem->mem_available_kb);
    printf("  \"mem_used_kb\": %ld,\n", mem->mem_total_kb - mem->mem_available_kb);
    printf("  \"swap_total_kb\": %ld,\n", mem->swap_total_kb);
    printf("  \"swap_free_kb\": %ld,\n", mem->swap_free_kb);
    printf("  \"load1\": %.2f,\n", load->load1);
    printf("  \"load5\": %.2f,\n", load->load5);
    printf("  \"load15\": %.2f\n", load->load15);
    printf("}\n");
}

int main(int argc, char *argv[]) {
    int json_mode = 0;
    if (argc > 1 && strcmp(argv[1], "--json") == 0) {
        json_mode = 1;
    }

    MemInfo mem;
    CpuInfo cpu;
    LoadInfo load;

    if (!read_mem_info(&mem)) {
        fprintf(stderr, "kacu-sysinfo: failed to read /proc/meminfo\n");
        return 1;
    }
    read_cpu_info(&cpu);   /* best effort, falls back to "unknown" */
    read_load_info(&load); /* best effort, falls back to zeros */

    if (json_mode) {
        print_json(&mem, &cpu, &load);
    } else {
        print_plain(&mem, &cpu, &load);
    }

    return 0;
}

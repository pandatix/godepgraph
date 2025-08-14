# Compiler and flags
CC = gcc
CFLAGS = -g -Wall -Wextra

# Target binary
TARGET = heaper_fort

# Source files
SRC = heaper-fort.c

# Build rules
all: $(TARGET)

$(TARGET): $(SRC)
	$(CC) $(CFLAGS) -o $(TARGET) $(SRC)

# Clean rule
clean:
	rm -f $(TARGET)

# Phony targets
.PHONY: all clean

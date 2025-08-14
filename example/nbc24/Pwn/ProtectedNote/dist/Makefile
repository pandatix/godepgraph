# Makefile for notes program

# Compiler and flags
ASM = nasm
LD = ld
ASMFLAGS = -f elf64
LDFLAGS =

# Files
TARGET = notes
OBJ = protected-notes.o
SRC = protected-notes.s

# Default target
all: $(TARGET)

# Rule to build the target
$(TARGET): $(OBJ)
	$(LD) $(LDFLAGS) -o $(TARGET) $(OBJ)

# Rule to assemble the source file
$(OBJ): $(SRC)
	$(ASM) $(ASMFLAGS) -o $(OBJ) $(SRC)

# Clean up object files and executable
clean:
	rm -f $(OBJ) $(TARGET)

# Run the program
run: $(TARGET)
	./$(TARGET)


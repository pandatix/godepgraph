%define PROT_READ 0x1
%define PROT_WRITE 0x2
%define PROT_EXEC 0x4

%define MAP_PRIVATE 0x2
%define MAP_ANONYMOUS 0x20


%define NOTE_SIZE 300
%define note_data_offset 0
%define note_canary_offset 256
%define note_size_offset 264
%define note_print_offset 272
    
; Stack NX
section .note.GNU-stack noalloc noexec nowrite progbits

section .data
    menu_text db 10,"Menu:", 10
    db "1. Create fancy note", 10
    db "2. Create reminder", 10
    db "3. Print note", 10
    db "4. Edit note", 10
    db "Enter choice >> ", 0

    note_msg db "Select the note you want to see", 10, 0
    enter_note_content_msg db "Enter note content >> ", 0
    slot_msg db "Enter slot [0-7] >> ", 0

    note_fancy_msg db "Here is your fancy note: ", 0
    note_reminder_msg db "Here is a reminder: ", 0

    create_msg db "[NEW NOTE CREATION]", 10, 0
    edit_msg db "[EDIT NOTE]", 10, 0

    error_slot_msg db "Error, invalid slot (0-7).", 10, 0
    error_note_empty_msg db "Error, note is empty.", 10, 0
    error_note_corrupted_msg db "Error, note is corrupted.", 10, 0

section .bss
    notes resq 8       ; *note [8]

section .text
    global _start

_start:
    jmp menu

exit:
    mov rax, 60            ; SYS_EXIT
    xor rdi, rdi           ; status 0
    syscall                ; invoke syscall

; ----------------------------------------
; ERRORS
; ----------------------------------------
error_invalid_slot:
    push error_slot_msg
    call print_string
    add rsp, 8

    jmp exit

error_note_empty:
    push error_note_empty_msg
    call print_string
    add rsp, 8

    jmp exit

error_note_corrupted:
    push error_note_corrupted_msg
    call print_string
    add rsp, 8

    jmp exit

; ----------------------------------------
; Function: menu
; ----------------------------------------
menu:
    ; Display the menu
    push menu_text          
    call print_string
    add rsp, 8

    ; Read user choice
    call read_choice

    cmp rax, '1'
    je fancy_note_create

    cmp rax, '2'
    je reminder_note_create

    cmp rax, '3'
    je print_note

    cmp rax, '4'
    je edit_note

    jmp exit


; ----------------------------------------
; Function: select_slot (Note * mmap_region)
;   returns selected note in rax
; ----------------------------------------
select_slot:
    push slot_msg
    call print_string
    add rsp, 8

    call read_choice                            ; select slot
    sub rax, 0x30

    cmp rax, 7
    jg error_invalid_slot

    cmp rax, 0
    jl error_invalid_slot

    lea rax, [notes+rax*8]    
    ret

note_fancy_print:
    push rbp
    mov rbp, rsp

    push note_fancy_msg
    call print_string
    add rsp, 8

    mov rax, 1        ; SYS_WRITE
    mov rdi, 1        ; fd (stdout)
    mov rsi, [rbp+24] ; buf*
    mov rdx, [rbp+16] ; size

    syscall

    leave
    ret

note_reminder_print:
    push rbp
    mov rbp, rsp

    push note_reminder_msg
    call print_string
    add rsp, 8

    mov rax, 1        ; SYS_WRITE
    mov rdi, 1        ; fd (stdout)
    mov rsi, [rbp+24] ; buf*
    mov rdx, [rbp+16] ; size

    syscall

    leave
    ret


; ----------------------------------------
; Function: print_note
; ----------------------------------------
print_note:
    push note_msg
    call print_string
    add rsp, 8

    call select_slot

    ; Test if note empty
    mov rsi, [rax]
    test rsi, rsi
    je error_note_empty

    ; Data buffer
    mov rsi, [rax]
    add rsi, note_data_offset
    push rsi

    ; Len
    mov rdx, [rax]
    add rdx, note_size_offset
    mov qword rdx, [rdx]
    push rdx

    ; Check canary
    mov rsi, [rax]
    add rsi, note_canary_offset
    mov rsi, [rsi]
    cmp rsi, 0x13371337
    jne error_note_corrupted

    ; print_func
    mov rax, [rax]
    add rax, note_print_offset
    
    mov rsi, rax ; dirty but usefull for the exploit :p
    call [rsi]

    add rsp, 16

    jmp menu

; ----------------------------------------
; Function: create_note
; ----------------------------------------
create_note:
    push rbp
    mov rbp, rsp

    push create_msg
    call print_string
    add rsp, 8

    ; Allocate new note with mmap
    mov r8, -1                                  ; fd (ignored but should be -1)
    mov rax, 9                                  ; SYS_MMAP
    mov rdi, 0                                  ; addr (operating system will choose the mapping destination)
    mov rsi, NOTE_SIZE                          ; length (size of the memory region to allocate)
    mov rdx, PROT_READ | PROT_WRITE | PROT_EXEC ; prot (make the new memory region RWX, we secure it afterwards anyway)
    mov r10, MAP_ANONYMOUS | MAP_PRIVATE        ; Not mapped to any file

    mov r9, 0                                   ; pgoff (page offset, typically 0 for anonymous mappings)
    syscall                                     ; invoke the system call, result in rax

    test rax, rax                               ; Check if mmap succeeded
    js exit                                     ; If mmap fails, go to exit
    push rax

    call select_slot
    pop rdi                                     ; new mmaped region
    mov [rax], rdi                              ; put pointer to new region in notes list

    mov rbx, rax

    ; fill note
    push enter_note_content_msg
    call print_string
    add rsp, 8

    xor rax, rax                                ; SYS_READ(unsigned int fd, char *buf, size_t count)
    xor rdi, rdi                                ; fd (stdin)
    mov rsi, [rbx]                              ; buf (new mapped memory region)
    add rsi, note_data_offset
    mov rdx, 256                                ; count (256)
    syscall

    mov rdi, [rbx]
    add rdi, note_canary_offset
    mov qword [rdi], 0x13371337                 ; int canary

    mov rdi, [rbx]
    add rdi, note_size_offset
    mov [rdi], rax                              ; int size

    mov rdi, [rbx]
    add rdi, note_print_offset
    mov rax, [rbp+16]                           ; get type of print
    mov [rdi], rax                              ; print_note *


    ; Set mprotect to disable execution on note
    mov rdx, PROT_READ                          ; prot (read_only now)
    mov rdi, [rbx]                              ; addr (new mapped memory region)
    mov rsi, NOTE_SIZE                          ; length (size of the memory region to allocate)
    mov rax, 10                                 ; SYS_MPROTECT
    syscall                                     ; invoke the system call, result in rax

    test rax, rax                               ; Check if mprotect succeeded
    js exit                                     ; If mprotect fails, go to exit


    xor rax, rax
    leave
    ret


fancy_note_create:
    lea rax, note_fancy_print
    push rax
    call create_note
    add rsp, 8
    jmp menu

reminder_note_create:
    lea rax, note_reminder_print
    push rax
    call create_note
    add rsp, 8
    jmp menu

; ----------------------------------------
; Function: edit_note
; ----------------------------------------
edit_note:
    push edit_msg
    call print_string
    add rsp, 8

    call select_slot
    mov rbx, rax

    push enter_note_content_msg
    call print_string
    add rsp, 8

    ; Unsecure note
    mov rdx, PROT_READ | PROT_WRITE             ; prot (rw)
    mov rdi, [rbx]                              ; addr (new mapped memory region)
    mov rsi, NOTE_SIZE                          ; length (size of the memory region to allocate)
    mov rax, 10                                 ; SYS_MPROTECT
    syscall                                     ; invoke the system call, result in rax

    test rax, rax                               ; Check if mprotect succeeded
    js exit                                     ; If mprotect fails, go to exit

    ; Get intput data
    xor rax, rax                                ; SYS_READ(unsigned int fd, char *buf, size_t count)
    xor rdi, rdi                                ; fd (stdin)
    mov rsi, [rbx]
    add rsi, note_data_offset                   ; buf
    mov rdx, 0x3000                             ; size, should be ok because there is canary
    syscall

    ; Secure note

    mov rdx, PROT_READ                          ; prot (read_only now)
    mov rdi, [rbx]                              ; addr (new mapped memory region)
    mov rsi, NOTE_SIZE                          ; length (size of the memory region to allocate)
    mov rax, 10                                 ; SYS_MPROTECT
    syscall                                     ; invoke the system call, result in rax

    test rax, rax                               ; Check if mprotect succeeded
    js exit                                     ; If mprotect fails, go to exit


    jmp menu

; ----------------------------------------
; Function: print_string
; print_string(char * str)
; ----------------------------------------
print_string:

    mov rsi, [rsp + 8]     ; char * buf

    xor rdx, rdx
.determine_lenstr:
    mov cl, byte [rsi + rdx]
    inc rdx
    test cl, cl             ; Check for null terminator
    jne .determine_lenstr

    mov rax, 1              ; SYS_WRITE(unsigned int fd, const char *buf, size_t count)
    mov rdi, 1              ; fd (stdout)
    syscall

    ret

; ----------------------------------------
; Function: read_choice
; ----------------------------------------
read_choice:
    mov rax, 0              ; syscall: read
    mov rdi, 0              ; file descriptor (stdin)
    sub rsp, 8
    mov rsi, rsp            ; Read input onto stack
    mov rdx, 3              ; Read one byte
    syscall
    movzx rax, byte [rsp]   ; Read choice
    add rsp, 8
    ret
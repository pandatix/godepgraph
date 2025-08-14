#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define BUF_SIZE 64

void setup() {
    setvbuf(stdout, NULL, _IONBF, 0);
    setvbuf(stderr, NULL, _IONBF, 0);
    setvbuf(stdin, NULL, _IONBF, 0);
}

typedef enum {
    TYPE_NONE,
    TYPE_WIZARD,
    TYPE_NECROMANCER,
    TYPE_SORCERER,
    TYPE_ALCHEMIST,
    TYPE_HUMAN
} CharacterType;

typedef struct Wizard {
    char name[BUF_SIZE];
    char pet_name[BUF_SIZE];
    void (*do_fly)();
    void (*do_magic)();
} Wizard;

typedef struct Necromancer {
    char name[BUF_SIZE];
    char spirit_name[BUF_SIZE];
    void (*invoke_spirit)();
} Necromancer;

typedef struct Sorcerer {
    char name[BUF_SIZE];
    void (*cast_spell)();
} Sorcerer;

typedef struct Alchemist {
    char name[BUF_SIZE];
    void (*mix_potion)();
} Alchemist;

typedef struct Human {
    char name[BUF_SIZE];
    int age;
} Human;

void fly() {
    printf("The wizard flies through the skies!\n");
}

void magic() {
    printf("The wizard casts a magical spell!\n");
}

void spirit() {
    printf("The necromancer invokes a spirit from beyond.\n");
}

void spell() {
    printf("The sorcerer casts a powerful spell.\n");
}

void potion() {
    printf("The alchemist brews a mysterious potion.\n");
}

void win() {
    printf("Congratz! You're the strongest!\n Here is your flag:");
    system("/bin/cat /flag.txt");
}

void menu() {
    printf("\n[Heap-er fort] Magical strongest\n\n");
    printf("1. Create character\n");
    printf("2. Delete character\n");
    printf("3. Perform action\n");
    printf("4. Exit\n");
    printf("Choice: ");
}

int main() {
    setup();

    void *characters[5] = {NULL};
    CharacterType slot_types[5] = {TYPE_NONE, TYPE_NONE, TYPE_NONE, TYPE_NONE, TYPE_NONE};
    int choice, index;

    while (1) {
        menu();
        scanf("%d", &choice);

        switch (choice) {
            case 1: // Create character
                printf("Choose a character slot (0-4): ");
                scanf("%d", &index);
                if (index < 0 || index >= 5) {
                    printf("Invalid slot.\n");
                    break;
                }
                if (characters[index] != NULL) {
                    printf("Slot already occupied.\n");
                    break;
                }
                printf("1. Wizard\n2. Necromancer\n3. Sorcerer\n4. Alchemist\n5. Human\nChoice: ");
                scanf("%d", &choice);
                if (choice == 1) {
                    Wizard *wizard = malloc(sizeof(Wizard));
                    printf("Enter wizard's name: ");
                    scanf("%s", wizard->name);
                    printf("Enter wizard's pet name: ");
                    scanf("%s", wizard->pet_name);
                    wizard->do_fly = fly;
                    wizard->do_magic = magic;
                    characters[index] = wizard;
                    slot_types[index] = TYPE_WIZARD;
                } else if (choice == 2) {
                    Necromancer *necromancer = malloc(sizeof(Necromancer));
                    printf("Enter necromancer's name: ");
                    scanf("%s", necromancer->name);
                    printf("Enter spirit name: ");
                    scanf("%s", necromancer->spirit_name);
                    necromancer->invoke_spirit = spirit;
                    characters[index] = necromancer;
                    slot_types[index] = TYPE_NECROMANCER;
                } else if (choice == 3) {
                    Sorcerer *sorcerer = malloc(sizeof(Sorcerer));
                    printf("Enter sorcerer's name: ");
                    scanf("%s", sorcerer->name);
                    sorcerer->cast_spell = spell;
                    characters[index] = sorcerer;
                    slot_types[index] = TYPE_SORCERER;
                } else if (choice == 4) {
                    Alchemist *alchemist = malloc(sizeof(Alchemist));
                    printf("Enter alchemist's name: ");
                    scanf("%s", alchemist->name);
                    alchemist->mix_potion = potion;
                    characters[index] = alchemist;
                    slot_types[index] = TYPE_ALCHEMIST;
                } else if (choice == 5) {
                    Human *human = malloc(sizeof(Human));
                    printf("Enter human's name: ");
                    scanf("%s", human->name);
                    printf("Enter human's age: ");
                    scanf("%d", &human->age);
                    characters[index] = human;
                    slot_types[index] = TYPE_HUMAN;
                } else {
                    printf("Invalid choice.\n");
                }
                break;

            case 2: // Delete character
                printf("Choose a character slot (0-4): ");
                scanf("%d", &index);
                if (index < 0 || index >= 5 || characters[index] == NULL) {
                    printf("Invalid slot.\n");
                    break;
                }
                free(characters[index]);
                printf("Character deleted.\n");
                break;

            case 3: // Perform action
                printf("Choose a character slot (0-4): ");
                scanf("%d", &index);
                if (index < 0 || index >= 5 || characters[index] == NULL) {
                    printf("Invalid slot.\n");
                    break;
                }

                switch (slot_types[index]) {
                    case TYPE_WIZARD: {
                        Wizard *wizard = (Wizard *)characters[index];
                        printf("1. Fly\n2. Magic\nChoice: ");
                        scanf("%d", &choice);
                        if (choice == 1) {
                            wizard->do_fly();
                        } else if (choice == 2) {
                            wizard->do_magic();
                        } else {
                            printf("Invalid choice.\n");
                        }
                        break;
                    }
                    case TYPE_NECROMANCER: {
                        Necromancer *necromancer = (Necromancer *)characters[index];
                        printf("1. Invoke Spirit\nChoice: ");
                        scanf("%d", &choice);
                        if (choice == 1) {
                            necromancer->invoke_spirit();
                        } else {
                            printf("Invalid choice.\n");
                        }
                        break;
                    }
                    case TYPE_SORCERER: {
                        Sorcerer *sorcerer = (Sorcerer *)characters[index];
                        printf("1. Cast Spell\nChoice: ");
                        scanf("%d", &choice);
                        if (choice == 1) {
                            sorcerer->cast_spell();
                        } else {
                            printf("Invalid choice.\n");
                        }
                        break;
                    }
                    case TYPE_ALCHEMIST: {
                        Alchemist *alchemist = (Alchemist *)characters[index];
                        printf("1. Mix Potion\nChoice: ");
                        scanf("%d", &choice);
                        if (choice == 1) {
                            alchemist->mix_potion();
                        } else {
                            printf("Invalid choice.\n");
                        }
                        break;
                    }
                    case TYPE_HUMAN: {
                        Human *human = (Human *)characters[index];
                        printf("Humans have no special abilities. Name: %s, Age: %d\n", human->name, human->age);
                        break;
                    }
                    default:
                        printf("Invalid character type.\n");
                        break;
                }
                break;

            case 4: // Exit
                printf("Goodbye!\n");
                return 0;

            default:
                printf("Invalid choice.\n");
        }
    }

    return 0;
}

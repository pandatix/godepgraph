use std::fs::{self};
use std::io::{self, Write};
use std::path::{PathBuf};
use std::sync::{Arc, Mutex};

// Main
fn main() {
    println!("Bienvenue sur le service Super-Cat. Vous pouvez selectionner un fichier et le lire plus tard ! Ça ne sert à rien ? Oui, mais c'est exploitable ;)");

    let selected_file: Arc<Mutex<Option<PathBuf>>> = Arc::new(Mutex::new(None));
    loop {
        println!("\nQue voulez-vous faire ?");
        println!("1. Sélectionner un fichier");
        println!("2. Informations sur le fichier");
        println!("3. Lire le fichier");
        println!("4. Quitter");

        let choice = read_input("Votre choix: ");
        match choice.trim() {
            "1" => select_file(selected_file.clone()),
            "2" => display_file_info(selected_file.clone()),
            "3" => read_file(selected_file.clone()),
            "4" => {
                println!("Bye bye. :)");
                break;
            }
            _ => println!("Choix non valide. x_x"),
        }
    }
}

fn read_input(prompt: &str) -> String {
    print!("{}", prompt);
    io::stdout().flush().unwrap();
    let mut input = String::new();
    io::stdin().read_line(&mut input).expect("Erreur lors de la lecture de l'entrée");
    input
}

fn select_file(selected_file: Arc<Mutex<Option<PathBuf>>>) {
    let path_str = read_input("Entrez le chemin du fichier: ");
    let path = PathBuf::from(path_str.trim());

    // Securité ! Vous ne pourrez pas lire le flag !!!!
    // Pas de symlink, pas de flag
    if path.is_symlink() || path.ends_with("flag.txt") {
        println!("Non, non, non ! Ce fichier est interdit !");
        return;
    }

    if path.exists() {
        *selected_file.lock().unwrap() = Some(path.clone());
        println!("Fichier sélectionné : {:?}", path);
    } else {
        println!("Le fichier n'existe pas. Veuillez réessayer.");
    }
}

fn display_file_info(selected_file: Arc<Mutex<Option<PathBuf>>>) {
    let file = selected_file.lock().unwrap();
    if let Some(path) = file.as_ref() {
        println!("Informations sur le fichier : {:?}", path);
    } else {
        println!("Aucun fichier sélectionné.");
    }
}

fn read_file(selected_file: Arc<Mutex<Option<PathBuf>>>) {
    let file = selected_file.lock().unwrap();
    if let Some(path) = file.as_ref() {
        match fs::read_to_string(path) {
            Ok(content) => println!("Contenu du fichier :\n{}", content),
            Err(err) => println!("Erreur lors de la lecture du fichier : {}", err),
        }
    } else {
        println!("Aucun fichier sélectionné.");
    }
}


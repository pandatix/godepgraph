from pathlib import Path

def merge_config(src, dst):
    for key, value in src.items():
        if hasattr(dst, '__getitem__'):
            if dst.get(key) and isinstance(value, dict):
                merge_config(value, dst.get(key))
            else:
                dst[key] = value
        elif hasattr(dst, key) and isinstance(value, dict):
            merge_config(value, getattr(dst, key))
        else:
            setattr(dst, key, value)

def safe_path_join(*paths):
    BASE_DIR = Path(__file__).resolve().parent
    final_path = BASE_DIR.joinpath(*paths).resolve()

    if BASE_DIR == final_path:
        raise ValueError("Écriture dans le répertoire de l'app interdite.")

    if BASE_DIR not in final_path.parents:
        raise ValueError("Tentative d'accès à un répertoire non autorisé")

    return final_path


from flask import Flask, redirect, request, render_template, make_response, send_from_directory, flash
from utils import merge_config, safe_path_join
from werkzeug.utils import secure_filename
from pathlib import Path
import os, jwt, json
from functools import wraps

app = Flask(__name__)
app.config['SECRET_KEY'] = 'iloveu'
app.config['TEMPLATES_AUTO_RELOAD'] = True

dev = False
path = 'uploads/'
allowed_extensions = {
    'image': ['jpg', 'jpeg', 'png'],
    'report': 'pdf',
    'usefull': 'json'
}

class ExtensionValidator:
    def __init__(self, allowed_extensions=None):
        if allowed_extensions is None:
            allowed_extensions = {}
        for key, value in allowed_extensions.items():
            setattr(self, key, value)

validator = ExtensionValidator(allowed_extensions)

def generate_token(username, role):
    token = jwt.encode({"username": username, "role": role}, app.config['SECRET_KEY'], algorithm="HS256")
    return token

def ensure_jwt(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        token = request.cookies.get('jwt_token')
        if not token:
            token = generate_token("guest_user", "user")
            response = make_response(f(*args, **kwargs))
            response.set_cookie('jwt_token', token, httponly=True)
            return response
        return f(*args, **kwargs)
    return decorated_function

def jwt_required_admin(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        token = request.cookies.get('jwt_token')
        if not token:
            flash("Token manquant", "error")
            return redirect('/')
        try:
            decoded_token = jwt.decode(token, app.config['SECRET_KEY'], algorithms=["HS256"])
            if decoded_token.get("role") != "admin":
                flash("Accès refusé, vous n'êtes pas admin", "error")
                return redirect('/')
        except jwt.InvalidTokenError:
            flash("Token invalide", "error")
            return redirect('/')
        return f(*args, **kwargs)
    return decorated_function

@app.route('/')
@ensure_jwt
def index():
    upload_dir = Path(__file__).resolve().parent / 'uploads'
    files = [f.name for f in upload_dir.glob("*") if f.is_file()]
    if not files:
        return render_template('index.html', files=[])    
    return render_template('index.html', files=files)

@app.route('/edit_extensions', methods=['GET', 'POST'])
@ensure_jwt
@jwt_required_admin
def edit_extensions():
    if request.method == 'GET':
        return render_template('edit_extensions.html')
    
    data = request.form.get('extensions')
    try:
        data = json.loads(data)
    except json.JSONDecodeError:
        flash("Format JSON invalide", "error")
        return redirect('/edit_extensions')

    merge_config(data, validator)
    flash("Configuration des extensions modifiées avec succès", "success")
    return redirect('/')

@app.route('/upload', methods=['GET', 'POST'])
@ensure_jwt
def upload_file():
    global path
    if request.method == 'GET':
        return render_template('upload.html')
    
    uploaded_file = request.files.get('file')
    if not uploaded_file:
        flash("Veuillez passer un fichier", "error")
        return redirect('/upload')
    
    filename = secure_filename(uploaded_file.filename)
    file_extension = filename.split('.')[-1]

    if not any(file_extension in getattr(validator, category, []) for category in vars(validator) if not category.startswith('__')):
        flash("Extension de fichier non autorisée", "error")
        return redirect('/upload')

    if not dev:
        filename = f"{filename}.txt"
    
    if not path:
        path = 'uploads/'

    try:
        save_path = safe_path_join(path, filename)
    except ValueError:
        flash("Chemin de destination non autorisé", "error")
        return redirect('/upload')

    existing_files = sorted(save_path.parent.glob("*"), key=lambda f: f.stat().st_ctime)
    if len(existing_files) >= 5:
        oldest_file = existing_files[0]
        oldest_file.unlink()

    os.makedirs(os.path.dirname(save_path), exist_ok=True)
    uploaded_file.save(save_path)
    return redirect('/')

@app.route('/download/<filename>', methods=['GET'])
@ensure_jwt
def download_file(filename):
    upload_dir = Path(__file__).resolve().parent / 'uploads'
    file_path = upload_dir / filename
    if not file_path.is_file():
        flash("Fichier introuvable", "error")
        return redirect('/')
    return send_from_directory(upload_dir, filename, as_attachment=True)

@app.route('/delete/<filename>', methods=['POST'])
@ensure_jwt
def delete_file(filename):
    upload_dir = Path(__file__).resolve().parent / 'uploads'
    
    file_path = upload_dir / filename
    if file_path.is_file():
        file_path.unlink()
        flash(f"Fichier '{filename}' supprimé avec succès", "success")
    else:
        flash("Fichier introuvable", "error")
    
    return redirect('/')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, threaded=True)
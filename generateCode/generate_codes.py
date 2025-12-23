import os
import string
import random
import requests
from dotenv import load_dotenv

load_dotenv()

SUPABASE_URL = os.getenv("SUPABASE_URL")
SUPABASE_KEY = os.getenv("SUPABASE_SERVICE_ROLE_KEY")

if not SUPABASE_URL or not SUPABASE_KEY:
    raise RuntimeError("SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY が未設定")

TABLE = "auth_codes"

HEADERS = {
    "apikey": SUPABASE_KEY,
    "Authorization": f"Bearer {SUPABASE_KEY}",
    "Content-Type": "application/json"
}

def generate_code(length=10):
    chars = string.ascii_uppercase + string.digits
    return "".join(random.choice(chars) for _ in range(length))

codes = []
for _ in range(50):
    codes.append({
        "code": generate_code(),
        "used": False
    })

res = requests.post(
    f"{SUPABASE_URL}/rest/v1/{TABLE}",
    headers=HEADERS,
    json=codes
)

if res.status_code not in (200, 201):
    print("❌ Insert failed:", res.text)
else:
    print("✅ 認証コード50件を登録しました")

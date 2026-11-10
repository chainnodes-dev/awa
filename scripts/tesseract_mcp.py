#!/usr/bin/env python3
"""
Tesseract OCR Model Context Protocol (MCP) Server for ASM Chain Nodes.
Provides fast, local, deterministic optical character recognition via tesseract CLI.
Runs over standard JSON-RPC 2.0 stdio transport with zero external dependencies.
"""

import sys
import os
import json
import subprocess
import glob
import tempfile

def resolve_file(file_path_or_id):
    """Resolves a file path, taking into account data/uploads directory structure."""
    if os.path.isabs(file_path_or_id) and os.path.exists(file_path_or_id):
        return file_path_or_id
    
    # Check relative to cwd
    if os.path.exists(file_path_or_id):
        return os.path.abspath(file_path_or_id)

    # Check data/uploads/<id>
    uploads_dir = os.path.join(os.getcwd(), "data", "uploads")
    candidate_dir = os.path.join(uploads_dir, file_path_or_id)
    if os.path.isdir(candidate_dir):
        files = [os.path.join(candidate_dir, f) for f in os.listdir(candidate_dir) if not f.startswith(".")]
        if files:
            return files[0]

    # Search in data/uploads recursively
    matches = glob.glob(os.path.join(uploads_dir, "**", file_path_or_id), recursive=True)
    if matches:
        return matches[0]

    return file_path_or_id

def run_tesseract(file_path, lang="deu+eng"):
    real_path = resolve_file(file_path)
    if not os.path.exists(real_path):
        return f"Error: file not found: {file_path}"
    
    # If PDF, convert pages to PNG images using pdftoppm first
    if real_path.lower().endswith(".pdf"):
        with tempfile.TemporaryDirectory() as tmpdir:
            prefix = os.path.join(tmpdir, "page")
            conv = subprocess.run(["pdftoppm", "-png", "-r", "150", real_path, prefix], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            if conv.returncode != 0:
                return f"PDF conversion error: {conv.stderr.strip()}"
            page_images = sorted(glob.glob(f"{prefix}-*.png"))
            if not page_images:
                return "Error: no pages converted from PDF"
            full_text = []
            for idx, pimg in enumerate(page_images, start=1):
                cmd = ["tesseract", pimg, "stdout", "-l", lang]
                proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)
                if proc.returncode == 0 and proc.stdout.strip():
                    full_text.append(f"--- Page {idx} ---\n" + proc.stdout.strip())
                elif proc.returncode != 0:
                    full_text.append(f"--- Page {idx} OCR error: {proc.stderr.strip()} ---")
            return "\n\n".join(full_text)

    cmd = ["tesseract", real_path, "stdout", "-l", lang]
    try:
        proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)
        if proc.returncode != 0:
            err = proc.stderr.strip()
            return f"Tesseract OCR Error (exit {proc.returncode}): {err}"
        return proc.stdout.strip()
    except Exception as e:
        return f"Failed to execute tesseract: {str(e)}"

TOOLS = [
    {
        "name": "ocr_image",
        "description": "Extract text from an image document (PNG, JPG, TIFF, WEBP, BMP) using Tesseract OCR with German & English support.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "image_path": {
                    "type": "string",
                    "description": "File path or filename of the image to process (e.g. upload file_id or absolute path)"
                },
                "lang": {
                    "type": "string",
                    "description": "Language code for recognition: 'deu', 'eng', or 'deu+eng' (default: 'deu+eng')",
                    "default": "deu+eng"
                }
            },
            "required": ["image_path"]
        }
    },
    {
        "name": "ocr_file",
        "description": "Extract text from an uploaded document (PDF or image) by resolving its blackboard file_id and running OCR.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "file_id": {
                    "type": "string",
                    "description": "The unique upload file_id from the workflow blackboard"
                },
                "lang": {
                    "type": "string",
                    "description": "Language code for recognition (default: 'deu+eng')",
                    "default": "deu+eng"
                }
            },
            "required": ["file_id"]
        }
    }
]

def handle_request(req):
    method = req.get("method")
    req_id = req.get("id")

    if method == "initialize":
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {}
                },
                "serverInfo": {
                    "name": "tesseract-ocr-server",
                    "version": "1.0.0"
                }
            }
        }
    
    elif method == "notifications/initialized":
        return None  # notification, no response
    
    elif method == "tools/list":
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "tools": TOOLS
            }
        }
    
    elif method == "tools/call":
        params = req.get("params", {})
        tool_name = params.get("name")
        args = params.get("arguments", {})

        lang = args.get("lang", "deu+eng")
        
        if tool_name == "ocr_image":
            img_path = args.get("image_path", "")
            text = run_tesseract(img_path, lang)
            return {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "content": [
                        {"type": "text", "text": text}
                    ]
                }
            }
        elif tool_name == "ocr_file":
            file_id = args.get("file_id", "")
            text = run_tesseract(file_id, lang)
            return {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "content": [
                        {"type": "text", "text": text}
                    ]
                }
            }
        else:
            return {
                "jsonrpc": "2.0",
                "id": req_id,
                "error": {
                    "code": -32601,
                    "message": f"Method or tool not found: {tool_name}"
                }
            }

    elif method == "ping":
        return {"jsonrpc": "2.0", "id": req_id, "result": {}}

    else:
        if req_id is not None:
            return {
                "jsonrpc": "2.0",
                "id": req_id,
                "error": {
                    "code": -32601,
                    "message": f"Method not found: {method}"
                }
            }
        return None

def main():
    while True:
        line = sys.stdin.readline()
        if not line:
            break
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
            resp = handle_request(req)
            if resp is not None:
                sys.stdout.write(json.dumps(resp) + "\n")
                sys.stdout.flush()
        except Exception as e:
            err_resp = {
                "jsonrpc": "2.0",
                "id": None,
                "error": {
                    "code": -32700,
                    "message": f"Parse error or exception: {str(e)}"
                }
            }
            sys.stdout.write(json.dumps(err_resp) + "\n")
            sys.stdout.flush()

if __name__ == "__main__":
    main()

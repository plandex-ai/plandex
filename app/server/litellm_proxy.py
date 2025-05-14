from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse, JSONResponse
from litellm import completion
import json
import re

LOGGING_ENABLED = False

print("Litellm proxy: starting proxy server on port 4000...")

app = FastAPI()

@app.get("/health")
async def health():
  return {"status": "ok"}

@app.post("/v1/chat/completions")
async def passthrough(request: Request):
  payload = await request.json()

  if LOGGING_ENABLED:
    # Log the request data for debugging
    try:
      # Get headers (excluding authorization to avoid logging credentials)
      headers = dict(request.headers)
      if "Authorization" in headers:
        headers["Authorization"] = "Bearer [REDACTED]"
      if "api-key" in headers:
        headers["api-key"] = "[REDACTED]"
      
      # Create a log-friendly representation
      request_data = {
        "method": request.method,
        "url": str(request.url),
        "headers": headers,
        "body": payload
      }
    
      # Log the request data
      print("Incoming request to /v1/chat/completions:")
      print(json.dumps(request_data, indent=2))
    except Exception as e:
      print(f"Error logging request: {str(e)}")

  model = payload.get("model", None)
  print(f"Litellm proxy: calling model: {model}")

  api_key = payload.pop("api_key", None)

  if not api_key:
    api_key = request.headers.get("Authorization")

  if not api_key:
    api_key = request.headers.get("api-key")

  if api_key and api_key.startswith("Bearer "):
    api_key = api_key.replace("Bearer ", "")

  # api key optional for local/ollama models, so no need to error if not provided

  try:
    if payload.get("stream"):
      def stream_generator():
        try:
          response_stream = completion(api_key=api_key, **payload)
          for chunk in response_stream:
            yield f"data: {json.dumps(chunk.to_dict())}\n\n"
          yield "data: [DONE]\n\n"
        except Exception as e:
          yield f"data: {json.dumps({'error': str(e)})}\n\n"

      print(f"Litellm proxy: Initiating streaming response for model: {payload.get('model', 'unknown')}")
      return StreamingResponse(stream_generator(), media_type="text/event-stream")

    else:
      print(f"Litellm proxy: Non-streaming response requested for model: {payload.get('model', 'unknown')}")
      result = completion(api_key=api_key, **payload)
      return JSONResponse(content=result)

  except Exception as e:
    err_msg = str(e)
    print(f"Litellm proxy: Error: {err_msg}")
    status_match = re.search(r"status code: (\d+)", err_msg)
    if status_match:
      status_code = int(status_match.group(1))
    else:
      status_code = 500
    return JSONResponse(
      status_code=status_code,
      content={"error": err_msg}
    )

from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse, JSONResponse
from litellm import completion, _turn_on_debug
import json
import re

# _turn_on_debug()

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

  # clean up for ollama if needed
  payload = normalise_for_ollama(payload)

  try:
    if payload.get("stream"):
      
      try:
        response_stream = completion(api_key=api_key, **payload)
      except Exception as e:
        return error_response(e)
      def stream_generator():
        try:  
          for chunk in response_stream:
            yield f"data: {json.dumps(chunk.to_dict())}\n\n"
          yield "data: [DONE]\n\n"
        except Exception as e:
          # surface the problem to the client _inside_ the SSE stream
          yield f"data: {json.dumps({'error': str(e)})}\n\n"
          return

        finally:
          try:
            response_stream.close()
          except AttributeError:
            pass

      print(f"Litellm proxy: Initiating streaming response for model: {payload.get('model', 'unknown')}")
      return StreamingResponse(stream_generator(), media_type="text/event-stream")

    else:
      print(f"Litellm proxy: Non-streaming response requested for model: {payload.get('model', 'unknown')}")
      try:
        result = completion(api_key=api_key, **payload)
      except Exception as e:
        return error_response(e)
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

def error_response(exc: Exception) -> JSONResponse:
  status = getattr(exc, "status_code", 500)
  retry_after = (
    getattr(getattr(exc, "response", None), "headers", {})
    .get("Retry-After")
  )
  hdrs = {"Retry-After": retry_after} if retry_after else {}
  return JSONResponse(status_code=status, content={"error": str(exc)}, headers=hdrs)

def normalise_for_ollama(p):
  if not p.get("model", "").startswith("ollama"):
    return p

  # flatten content parts
  for m in p.get("messages", []):
    if isinstance(m["content"], list):  # [{type:"text", text:"…"}]
        m["content"] = "".join(part.get("text", "")
                                for part in m["content"]
                                if part.get("type") == "text")

  # drop params Ollama ignores
  for k in ("top_p", "temperature", "presence_penalty",
            "tool_choice", "tools", "seed"):
      p.pop(k, None)

  return p
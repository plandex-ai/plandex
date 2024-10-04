import json

def convert_to_json(data):
    try:
        json_string = json.dumps(data)
        return json_string
    except TypeError as e:
        return str(e)
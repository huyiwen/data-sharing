from typing import Tuple
from flask import Flask, jsonify, request, render_template, send_from_directory
from flask.helpers import make_response
from flask_cors import CORS
import sys

app = Flask(__name__)
CORS(app)
eprint = lambda *args, **kwargs: print(*args, file=sys.stderr, **kwargs)


@app.route('/get_services', methods=['POST'])
def get_services():
    services = [
        {
            "serviceName": "Service1",
            "serviceID": "ID001",
            "sellerURL": "http://seller1.example.com",
            "sellerPublicKey": "abc123PublicKey",
            "comment": "Excellent service!",
            "sellerHeaders": {
                "Authorization": "Bearer abc123",
                "Content-Type": "application/json"
            },
            "transactionHash": "0x1234567892",
        },
        {
            "serviceName": "Service2",
            "serviceID": "ID002",
            "sellerURL": "http://seller2.example.com",
            "sellerPublicKey": "def456PublicKey",
            "comment": "Reliable and efficient.",
            "sellerHeaders": {
                "Authorization": "Bearer def456",
                "Content-Type": "application/json"
            },
            "transactionHash": "0x1234567891",
        },
        {
            "serviceName": "Service3",
            "serviceID": "ID003",
            "sellerURL": "http://seller3.example.com",
            "sellerPublicKey": "ghi789PublicKey",
            "comment": "Good, but room for improvement.",
            "sellerHeaders": {
                "Authorization": "Bearer ghi789",
                "Content-Type": "application/json"
            },
            "transactionHash": "0x1234567890",
        }
    ]
    return jsonify({"services": services}), 200

@app.route('/approve_application', methods=['POST'])
def approve_application():
    data = request.get_json()
    eprint(data)
    return jsonify({
        "serviceID": "ID005"
    }), 200

@app.route('/put_service', methods=['POST'])
def put_service():
    data = request.get_json()
    eprint(data)
    return jsonify({
        "serviceID": "ID005",
        "transactionHash": "0x123468q235"
    }), 200

@app.route('/fetch_data', methods=['POST'])
def fetch_data():
    data = request.get_json()
    eprint(data)
    return jsonify({
        "data": [[1, 2, 3], [4, 5, 6], [7, 8, 9]],
        "column_names": ["column1", "column2", "column3"],
    }), 200

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/app.js')
def scripts():
    return send_from_directory('..', "app.js")

@app.route('/styles.css')
def styles():
    return send_from_directory('..', "styles.css")

if __name__ == "__main__":
    app.run(debug=True)

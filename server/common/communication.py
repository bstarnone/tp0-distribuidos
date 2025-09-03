import struct


def consume_socket_data(client_sock):
    msg_type = client_sock.recv(1)
    raw_len = client_sock.recv(2) #TODO poner una constante
    print(f"leo raw len {raw_len}")
    if not raw_len:
        return -1, -1
    msg_len = (raw_len[0] << 8) | raw_len[1]
    msg = b""
    while len(msg) < msg_len: #evitando short-reads
        data_rcv = client_sock.recv(msg_len - len(msg))
        if not data_rcv:
            break
        msg += data_rcv
    return msg_type[0], msg


def send_client_bet_response(client_sock, dni: str, num: str):
    msg = f"{dni};{num}"
    msg_bytes = msg.encode("utf-8")
    msg_len = len(msg_bytes)
    buf = bytes([
        (msg_len >> 8) & 0xFF,
        msg_len & 0xFF
    ])

    client_sock.sendall(buf)
    print("DEBUG-ENVIO ", msg_len, " bytes")
    client_sock.sendall(msg_bytes)
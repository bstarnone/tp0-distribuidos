import struct


def consume_socket_data(client_sock):
    raw_len = client_sock.recv(2) #TODO poner una constante
    # print(f"leo raw len {raw_len}")
    if not raw_len:
        return -1
    msg_len = struct.unpack(">H", raw_len)[0] #usa BigEndian (>I) para convertir los 2 bytes a un entero
    msg = b""
    while len(msg) < msg_len: #evitando short-reads
        data_rcv = client_sock.recv(msg_len - len(msg))
        if not data_rcv:
            break
        msg += data_rcv
    return msg


def send_client_bet_response(client_sock, dni: str, num: str):
    msg = f"{dni};{num}"
    msg_bytes = msg.encode("utf-8")
    msg_len = len(msg_bytes)

    client_sock.sendall(struct.pack(">I", msg_len))
    print("DEBUG-ENVIO ", msg_len, " bytes")
    client_sock.sendall(msg_bytes)
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


def send_client_winners(client_sock, winners):
    parts = []
    for w in winners:
        parts.append(f"{w.document};{w.number}")

    # unir con coma sin la última extra
    payload_str = ",".join(parts)

    payload_bytes = payload_str.encode("utf-8")
    msg_len = len(payload_bytes)
    len_bytes = bytes([(msg_len >> 8) & 0xFF, msg_len & 0xFF])

    buf = bytearray(1 + 2 + msg_len)
    buf[0] = 3
    buf[1:3] = len_bytes
    buf[3:] = payload_bytes

    client_sock.sendall(buf)
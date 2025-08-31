import struct


def consume_socket_data(client_sock):
    raw_len = client_sock.recv(4) #TODO poner una constante
    msg_len = struct.unpack(">I", raw_len)[0] #usa BigEndian (>I) para convertir los 4 bytes a un entero
    msg = b""

    while len(msg) < msg_len: #evitando short-reads
        data_rcv = client_sock.recv(msg_len - len(msg))
        if not data_rcv:
            break
        msg += data_rcv
    return msg
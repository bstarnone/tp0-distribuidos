import socket
import logging
import struct
import sys

from . import communication as comms
from . import utils

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.client_sockets = []

    def sigterm_handler(self, signum=None, frame=None):
        self.free_resources()
        logging.info(f'action: shutdown | result: success')
        sys.exit(0)

    def free_resources(self):
        logging.info(f'action: close server socket | in_progress')
        self._server_socket.close()
        logging.info(f'action: close server socket | success')
        logging.info(f'action: close clients sockets | in_progress')

        for client_socket in self.client_sockets:
            client_socket.close()
        logging.info(f'action: close clients sockets | success')

    def getBetsFromBytes(self, msg):
        raw_bets = msg.rstrip().decode('utf-8').split(',')
        bets = []
        for raw_bet in raw_bets:
            # print(f"[RAW-BET] {raw_bet}")
            parsed_bet = raw_bet.split(';')
            # print(f"[SPLITED-BET] {parsed_bet}")
            bet = utils.Bet(
                parsed_bet[0],
                parsed_bet[1],
                parsed_bet[2],
                parsed_bet[3],
                parsed_bet[4],
                parsed_bet[5]
            )
            bets.append(bet)
        return bets

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        while True:
            client_sock = self.__accept_new_connection()
            self.client_sockets.append(client_sock)
            self.__handle_client_connection(client_sock)

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            stored_bets=0
            while True:
                msg = comms.consume_socket_data(client_sock)
                # print(f"consumo del cliente: {msg}")
                if msg == -1: #nada para consumir
                    break
                # msg_split = msg.rstrip().decode('utf-8').split(';')
                # bet = utils.Bet(
                #     msg_split[0],
                #     msg_split[1],
                #     msg_split[2],
                #     msg_split[3],
                #     msg_split[4],
                #     msg_split[5]
                # )
                # utils.store_bets([bet])
                bets = self.getBetsFromBytes(msg)
                utils.store_bets(bets)
                stored_bets += len(bets)
                # logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')
                # comms.send_client_bet_response(client_sock, bet.document, bet.number)
                logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
        except OSError as e:
            logging.error(f'action: apuesta_recibida | result: fail | cantidad: {stored_bets}')
        finally:
            logging.info(f'action: apuesta_recibida | result: success | cantidad: {stored_bets}')
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

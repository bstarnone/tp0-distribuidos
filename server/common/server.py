import socket
import logging
import struct
import sys
import threading

from . import communication as comms
from . import utils

class Client:
    def __init__(self, socket):
        self.socket = socket
        self.agency_id = 0
        self.bets = []
        self.finished = False
        self.winners = []
        self.asked_winners = False

class Server:
    def __init__(self, port, listen_backlog, expected_agencies):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.expected_agencies = expected_agencies
        # self.client_sockets = []
        self.clients = []
        # self.winners = []
        self.winners_response_queue = []
        self.lock = threading.Lock()
        self.threads = []
        self.barrier = threading.Barrier(expected_agencies)

    def sigterm_handler(self, signum=None, frame=None):
        self.free_resources()
        logging.info(f'action: shutdown | result: success')
        sys.exit(0)

    def free_resources(self):
        logging.info(f'action: close server socket | in_progress')
        self._server_socket.close()
        logging.info(f'action: close server socket | success')
        logging.info(f'action: close clients sockets | in_progress')
        for t in self.threads:
            t.join()

        for client in self.clients:
            client.socket.close()
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

        return bets, bets[0].agency

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
            client = Client(client_sock)
            with self.lock:
                self.clients.append(client)

            t = threading.Thread(
                target=self.__handle_client_connection,
                args=(client,),
            )
            t.start()
            self.threads.append(t)

            # self.__handle_client_connection(client)

    def handle_message(self, client, msg_type, payload): #TODO deberia ser un mensaje de Client
        # print(f"Escuchando: {payload}")

        stored_bets=0

        if msg_type == 1: #batch apuestas
            bets, agency_id = self.getBetsFromBytes(payload)
            client.agency_id = agency_id
            with self.lock:
                utils.store_bets(bets)
            stored_bets += len(bets)
            # logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
        if msg_type == 2: #fin batch
            client.finished = True
            print("fin batch")
        if msg_type == 3: #pide ganador
            print("quiero el ganador")
            client.asked_winners = True
            # print(client.winners)
            # comms.send_client_winners(client.socket, client.winners)
            self.winners_response_queue.append(client)
        return stored_bets

    def find_every_winner(self, client):
        with self.lock:
            all_bets = utils.load_bets()
            for bet in all_bets:
                if utils.has_won(bet):
                    if bet.agency == client.agency_id:
                            print(f"cliente {client.agency_id} suma ganador {bet.document}")
                            client.winners.append(bet)

    def __handle_client_connection(self, client): #TODO deberia recibir todo el client
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            stored_bets=0
            while True:
                msg_type, msg = comms.consume_socket_data(client.socket)

                if msg == -1: #nada para consumir
                    break

                stored_bets += self.handle_message(client, msg_type, msg)

                # if len(self.winners_response_queue) == self.expected_agencies:
                #     self.find_every_winner()
                #     for c in self.clients:
                #         if len(c.winners) > 0:
                #             comms.send_client_winners(c.socket, c.winners)
                #         c.socket.close()
                #     break
                if client.finished & client.asked_winners:
                    break
        except OSError as e:
            logging.error(f'action: apuesta_recibida | result: fail | cantidad: {stored_bets} {e}')
        finally:
            logging.info(f'action: apuesta_recibida | result: success | cantidad: {stored_bets}')
            print(f"Llego a la barrier")
            self.barrier.wait()
            print(f"Pasé la barrier")

        if len(self.winners_response_queue) == self.expected_agencies:
            self.find_every_winner(client)
            if len(client.winners) > 0:
                comms.send_client_winners(client.socket, client.winners)
            client.socket.close()

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

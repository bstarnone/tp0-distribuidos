# Comentarios sobre la resolución
## Sobre los cambios solicitados (reentrega)
En esta sección dejo un breve detalle de las correcciones solicitadas por Máximo para el TP0.
### Cambio en el manejo de threads
Los threads al terminar son joineados para liberar los recursos, reparando el caso en que un cliente envíe sus apuestas y no deje esperando la conexión y todos los recursos que conlleva hasta que se define un ganador.
Se implementó un thread que verifica, periódicamente (con un sleep para evitar un busy wait), si algún thread de los que manejan clientes terminó su ejecución.
```python
    t = threading.Thread( #thread para joinear los que terminen y liberar recursos
        target=self.check_joinable_threads,
    )
    t.start()
    self.cleaner_thread = t

  def check_joinable_threads(self):
      while True:
          time.sleep(0.5)
          with self.lock:
              alive_threads_copy = self.threads[:]
              for t in alive_threads_copy:
                  if not t.is_alive():
                      t.join()
                      self.threads.remove(t)
```

### Evitar short-read en el cliente
La manera original de leer respuestas del servidor en el cliente utiliza ReadAll() que devuelve un error != nil si no pudo leer la cantidad de bytes que esperaba. El detalle es que no existía un reintento de la lectura si fallaba. Ahora se agrega el reintento si la recepción de los ganadores falla.
```go
response, _ := receiveWinners(c.conn)

n := 1 * time.Second
for response == -1 {
    log.Infof("action: consulta_ganadores | result: in progress | status: awaiting winners finish")
    time.Sleep(n)
    response, _ = receiveWinners(c.conn)
    n = n * 2
}
```
Y la función `receiverWinners()` fue actualizada para reintentar o eventualmente fallar si hay un short-read. De esta manera si hay un fallo recibiendo datos desde el servidor, se vuelve a hacer el pedido.


## Ejercicio 1
Para este ejercicio se realizó un script `generar_compose.sh` únicamente con bash ya que se considera que es suficiente para lo solicitado.
El script toma como variables de entrada el nombre del archivo de salida y la cantidad de clientes a generar.
Internamente abre el archivo, escribe la parte del servidor, realiza un loop generando la parte para los clientes en función de cuántos clientes hayan y finalmente escribe la parte de la red. El modelo utilizado fue el `docker-compose-dev.yaml` provisto en el repositorio base.
Este script fue evolucionando entre los ejercicios, así que pueden existir diferencias entre las distintas ramas.

Puede invocarse mediante: `bash generar_compose.sh <output_file_name> <#clients>`

Ejemplo: `bash generar_compose.sh docker_compose_dev.yml 5`

## Ejercicio 2
Este ejercicio pide que se pueda configurar de manera dinámica y externa a los containers su configuración (es decir, `config.ini` para el server y `config.yaml` para los clientes). Esto se logra usando *docker volumes* y agregandolos al compose del ejercicio anterior.
Analizando los dockerfiles de cada aplicación, se determinó que los archivos viven en el root del contenedor, dado que allí se copian inicialmente.
Entonces se agregan al compose modificando el script:
- Para el cliente:
```
volumes:
  - ./client/config.yaml:/config.yaml
```

- Para el servidor:
```
volumes:
  - ./server/config.ini:/config.ini
```

## Ejercicio 3
Este ejercicio pide constatar si el servidor está levantado utilizando `netcat`, con la particularidad que el comando no debe correrse desde el host que lo ejecuta.
Por esta limitación, el script lo corre usando una imagen docker:
```
docker run --rm --network=tp0_testing_net alpine /bin/sh -c 'echo "mensaje" | nc server 12345'
```
El comando en cuestión levanta una imagen de `alpine`, que se verificó previamente que venga con `netcat` listo para su uso.
Se lo corre conectandolo a la red interna de docker que se crea en el script `generar_compose.sh` y se le pasa el comando que utiliza `netcat` para que pueda ejecutar la verificación.
El script luego captura la respuesta y verifica si el mensaje coincide con lo enviado, imprimiendo el mensaje correspondiente.

## Ejercicio 4
Este ejercicio pide implementar un *graceful shutdown* cuando las aplicaciones reciben la señal SIGTERM.
La flag `-t`, utilizada en el make file para matar los contenedores, indica cuánto tiempo (en segundos) se le da a los contenedores para cerrarse desde que se envía SIGTERM. Pasado ese tiempo, envía SIGKILL.
Para implementar el manejo de señales, en el servidor se utilizó el módulo `signal`:
```python
    signal.signal(signal.SIGTERM, server.sigterm_handler)
```
Esta función recibe la señal que debe manejar y la función para manejarla. La función se define dentro de `server.py` como:
```python
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

```
Y se encarga de asegurarse que todos los sockets abiertos para la comunicación se cierren a fin de evitar que queden *file descriptors* sueltos.

Para el cliente la cuestión es análoga pero utilizando *channels* de Go.
```go
  sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
```
En este caso, esta implementación no provee de un handler, sino que se revisa dentro del loop del cliente.
```go
func (c *Client) StartClientLoop(sigChan chan os.Signal) {
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		select {
		case <-sigChan:
			c.conn.Close()
			log.Infof("action: shutdown | result: success")
			return
		default:
    ///...
```
Y si la señal se dispara, la conexión con el servidor se cierra.

## Ejercicio 5
El protocolo de comunicación utilizado para este ejercicio consiste en que el emisor da a conocer al receptor la cantidad de bytes que va a mandar antes de enviarlos:
- El emisor envía 4 bytes con la cantidad de bytes que ocupa la información codificada.
- Luego envía la información codificada con un separador ";"
- El receptor recibe la cantidad de bytes y luego esperar a leer los mismos desde el socket
- Una vez recibidos los datos, los puede decodificar utilizando el separador ";".
---
El formato de un mensaje típico de este protocolo se ve así: <br>
- mensaje1: `<4 bytes - largo paquete>` <br>
- mensaje2: `<id_agencia>;<nombre>;<apellido>;<DNI>;<nacimiento>;<número apostado>` <br>

---
Para manejar los short-reads y short-writes la idea general, compartida a lo largo del proyecto, es la de contar cuántos bytes se envían y compararlos con la cantidad de bytes que se quieren enviar.

En el cliente se implementó así la escritura evitando short-writes:
```go
	for total_sent < len(full_msg) {
		n, err := connection.Write(full_msg[total_sent:])
		if err != nil {
			fmt.Println("[ERROR] Error enviando bet al servidor:", err)
			return
		}
		total_sent += n
	}
}
```

Y análogamente en el servidor se implementa la lectura evitando short-reads:
```python
   msg = b""

  while len(msg) < msg_len: #evitando short-reads
        data_rcv = client_sock.recv(msg_len - len(msg))
        if not data_rcv:
            break
        msg += data_rcv
  return msg
```
## Ejercicio 6
Este ejercicio pide la implementación de envío por batches. Dado que el envío de batches requiere la serialización de muchas apuestas juntas en un mismo mensaje, el protocolo para el ejercicio 5 se queda corto ya que solamente serializa de a una apuesta y la envía.
Los cambios propuestos pasan a ser:
- `<2 bytes>` para el largo del paquete total (como las batches no pueden ser mayores a 8kB, 2 bytes alcanza bien para representar la cantidad de bytes enviados)
- `<apuesta1>,<apuesta2>,...` las apuestas se serializan de la siguiente manera, separadas por ",".
- `<id_agencia>;<nombre>;<apellido>;<DNI>;<nacimiento>;<número apostado>` y los valores de cada apuesta se mantienen de la misma manera que el ejercicio 5.

<br>De esta manera, un mensaje podría ser:
<br>`<N bytes>,1;Juan;Perez;4440000;1999-05-03;4444,1;Pepe;Gomez;55555000;2003-06-07,5555`

### Pruebas para batchsize
Utilizando el dataset `dataset-1.csv` provisto por la cátedra se corrieron varias veces el sistema para determinar el número óptimo de batchSize para que el programa envíe las batchs más grandes sin superar los 8kB.
- Con 5 anduvo bien
- Con 50 también
- Con 500 ya no, la batch llegaba a ~23kB
- Con 23k/8kB = 2,875 => habría que reducir en aprox 1/3 el batchSize, 500/3 =~167
- Con 167 funciona bien
- Con 200 la batch queda de 9kB aproximadamente
- Finalmente, como 167 funcionó bien y es una estimación conservadora, ese va a ser el valor final propuesto.

## Ejercicio 7
Dado que se solicita soporte para recibir diferentes tipos de mensajes (batch, fin de batch y consulta de ganadores), es necesario revisar nuevamente el protocolo de comunicación.
Simplemente se agrega un nuevo header de 1 byte para el tipo siendo, en decimal: 1 para batch, 2 para fin y 3 para consultar ganadores.
```
+-----------+----------------+--------------------+
| msg_type  | length_prefix  | payload            |
| 1 byte    | 2 bytes        | N bytes            |
+-----------+----------------+--------------------+
```

Se actualizó el programa para que el servidor reciba todas las apuestas, espere a que cada cliente le envíe el pedido de ganadores y ahí los envía analizando el archivo y enviando solamente a los que corresponde. Cada cliente envía la solicitud de ganadores y se queda esperando por una respuesta hasta que el servidor la envía.

## Ejercicio 8
Para la concurrencia se optó por el uso de *multithreading*. Se utiliza debido a que es una librería con conceptos familiares sobre concurrencia, provee la creación de threads para paralelizar y el uso de barreras para la sincronización. Además, leyendo acerca de GIL y la manera que tiene Python de manejar el paralelismo, se observa que no se aprovechan las capacidades multiprocesador de la computadora. Igualmente, el trabajo práctico no es de naturaleza *CPU-Intensive*, sino que es *I/O intensive* tanto por la comunicación entre procesos como por la escritura y lectura de archivos. En conclusión, el no aprovechar todo el CPU no es algo que impacte en lo central del problema a resolver y sí se aprovechas las herramientas y la familiaridad de la biblioteca para implementar el paralelismo solicitado.

# TP0: Docker + Comunicaciones + Concurrencia

En el presente repositorio se provee un esqueleto básico de cliente/servidor, en donde todas las dependencias del mismo se encuentran encapsuladas en containers. Los alumnos deberán resolver una guía de ejercicios incrementales, teniendo en cuenta las condiciones de entrega descritas al final de este enunciado.

 El cliente (Golang) y el servidor (Python) fueron desarrollados en diferentes lenguajes simplemente para mostrar cómo dos lenguajes de programación pueden convivir en el mismo proyecto con la ayuda de containers, en este caso utilizando [Docker Compose](https://docs.docker.com/compose/).

## Instrucciones de uso
El repositorio cuenta con un **Makefile** que incluye distintos comandos en forma de targets. Los targets se ejecutan mediante la invocación de:  **make \<target\>**. Los target imprescindibles para iniciar y detener el sistema son **docker-compose-up** y **docker-compose-down**, siendo los restantes targets de utilidad para el proceso de depuración.

Los targets disponibles son:

| target  | accion  |
|---|---|
|  `docker-compose-up`  | Inicializa el ambiente de desarrollo. Construye las imágenes del cliente y el servidor, inicializa los recursos a utilizar (volúmenes, redes, etc) e inicia los propios containers. |
| `docker-compose-down`  | Ejecuta `docker-compose stop` para detener los containers asociados al compose y luego  `docker-compose down` para destruir todos los recursos asociados al proyecto que fueron inicializados. Se recomienda ejecutar este comando al finalizar cada ejecución para evitar que el disco de la máquina host se llene de versiones de desarrollo y recursos sin liberar. |
|  `docker-compose-logs` | Permite ver los logs actuales del proyecto. Acompañar con `grep` para lograr ver mensajes de una aplicación específica dentro del compose. |
| `docker-image`  | Construye las imágenes a ser utilizadas tanto en el servidor como en el cliente. Este target es utilizado por **docker-compose-up**, por lo cual se lo puede utilizar para probar nuevos cambios en las imágenes antes de arrancar el proyecto. |
| `build` | Compila la aplicación cliente para ejecución en el _host_ en lugar de en Docker. De este modo la compilación es mucho más veloz, pero requiere contar con todo el entorno de Golang y Python instalados en la máquina _host_. |

### Servidor

Se trata de un "echo server", en donde los mensajes recibidos por el cliente se responden inmediatamente y sin alterar.

Se ejecutan en bucle las siguientes etapas:

1. Servidor acepta una nueva conexión.
2. Servidor recibe mensaje del cliente y procede a responder el mismo.
3. Servidor desconecta al cliente.
4. Servidor retorna al paso 1.


### Cliente
 se conecta reiteradas veces al servidor y envía mensajes de la siguiente forma:

1. Cliente se conecta al servidor.
2. Cliente genera mensaje incremental.
3. Cliente envía mensaje al servidor y espera mensaje de respuesta.
4. Servidor responde al mensaje.
5. Servidor desconecta al cliente.
6. Cliente verifica si aún debe enviar un mensaje y si es así, vuelve al paso 2.

### Ejemplo

Al ejecutar el comando `make docker-compose-up`  y luego  `make docker-compose-logs`, se observan los siguientes logs:

```
client1  | 2024-08-21 22:11:15 INFO     action: config | result: success | client_id: 1 | server_address: server:12345 | loop_amount: 5 | loop_period: 5s | log_level: DEBUG
client1  | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:14 DEBUG    action: config | result: success | port: 12345 | listen_backlog: 5 | logging_level: DEBUG
server   | 2024-08-21 22:11:14 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°3
client1  | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°3
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°5
client1  | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°5
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:40 INFO     action: loop_finished | result: success | client_id: 1
client1 exited with code 0
```


## Parte 1: Introducción a Docker
En esta primera parte del trabajo práctico se plantean una serie de ejercicios que sirven para introducir las herramientas básicas de Docker que se utilizarán a lo largo de la materia. El entendimiento de las mismas será crucial para el desarrollo de los próximos TPs.

### Ejercicio N°1:
Definir un script de bash `generar-compose.sh` que permita crear una definición de Docker Compose con una cantidad configurable de clientes.  El nombre de los containers deberá seguir el formato propuesto: client1, client2, client3, etc.

El script deberá ubicarse en la raíz del proyecto y recibirá por parámetro el nombre del archivo de salida y la cantidad de clientes esperados:

`./generar-compose.sh docker-compose-dev.yaml 5`

Considerar que en el contenido del script pueden invocar un subscript de Go o Python:

```
#!/bin/bash
echo "Nombre del archivo de salida: $1"
echo "Cantidad de clientes: $2"
python3 mi-generador.py $1 $2
```

En el archivo de Docker Compose de salida se pueden definir volúmenes, variables de entorno y redes con libertad, pero recordar actualizar este script cuando se modifiquen tales definiciones en los sucesivos ejercicios.

### Ejercicio N°2:
Modificar el cliente y el servidor para lograr que realizar cambios en el archivo de configuración no requiera reconstruír las imágenes de Docker para que los mismos sean efectivos. La configuración a través del archivo correspondiente (`config.ini` y `config.yaml`, dependiendo de la aplicación) debe ser inyectada en el container y persistida por fuera de la imagen (hint: `docker volumes`).


### Ejercicio N°3:
Crear un script de bash `validar-echo-server.sh` que permita verificar el correcto funcionamiento del servidor utilizando el comando `netcat` para interactuar con el mismo. Dado que el servidor es un echo server, se debe enviar un mensaje al servidor y esperar recibir el mismo mensaje enviado.

En caso de que la validación sea exitosa imprimir: `action: test_echo_server | result: success`, de lo contrario imprimir:`action: test_echo_server | result: fail`.

El script deberá ubicarse en la raíz del proyecto. Netcat no debe ser instalado en la máquina _host_ y no se pueden exponer puertos del servidor para realizar la comunicación (hint: `docker network`). `


### Ejercicio N°4:
Modificar servidor y cliente para que ambos sistemas terminen de forma _graceful_ al recibir la signal SIGTERM. Terminar la aplicación de forma _graceful_ implica que todos los _file descriptors_ (entre los que se encuentran archivos, sockets, threads y procesos) deben cerrarse correctamente antes que el thread de la aplicación principal muera. Loguear mensajes en el cierre de cada recurso (hint: Verificar que hace el flag `-t` utilizado en el comando `docker compose down`).

## Parte 2: Repaso de Comunicaciones

Las secciones de repaso del trabajo práctico plantean un caso de uso denominado **Lotería Nacional**. Para la resolución de las mismas deberá utilizarse como base el código fuente provisto en la primera parte, con las modificaciones agregadas en el ejercicio 4.

### Ejercicio N°5:
Modificar la lógica de negocio tanto de los clientes como del servidor para nuestro nuevo caso de uso.

#### Cliente
Emulará a una _agencia de quiniela_ que participa del proyecto. Existen 5 agencias. Deberán recibir como variables de entorno los campos que representan la apuesta de una persona: nombre, apellido, DNI, nacimiento, numero apostado (en adelante 'número'). Ej.: `NOMBRE=Santiago Lionel`, `APELLIDO=Lorca`, `DOCUMENTO=30904465`, `NACIMIENTO=1999-03-17` y `NUMERO=7574` respectivamente.

Los campos deben enviarse al servidor para dejar registro de la apuesta. Al recibir la confirmación del servidor se debe imprimir por log: `action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}`.



#### Servidor
Emulará a la _central de Lotería Nacional_. Deberá recibir los campos de la cada apuesta desde los clientes y almacenar la información mediante la función `store_bet(...)` para control futuro de ganadores. La función `store_bet(...)` es provista por la cátedra y no podrá ser modificada por el alumno.
Al persistir se debe imprimir por log: `action: apuesta_almacenada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Comunicación:
Se deberá implementar un módulo de comunicación entre el cliente y el servidor donde se maneje el envío y la recepción de los paquetes, el cual se espera que contemple:
* Definición de un protocolo para el envío de los mensajes.
* Serialización de los datos.
* Correcta separación de responsabilidades entre modelo de dominio y capa de comunicación.
* Correcto empleo de sockets, incluyendo manejo de errores y evitando los fenómenos conocidos como [_short read y short write_](https://cs61.seas.harvard.edu/site/2018/FileDescriptors/).


### Ejercicio N°6:
Modificar los clientes para que envíen varias apuestas a la vez (modalidad conocida como procesamiento por _chunks_ o _batchs_).
Los _batchs_ permiten que el cliente registre varias apuestas en una misma consulta, acortando tiempos de transmisión y procesamiento.

La información de cada agencia será simulada por la ingesta de su archivo numerado correspondiente, provisto por la cátedra dentro de `.data/datasets.zip`.
Los archivos deberán ser inyectados en los containers correspondientes y persistido por fuera de la imagen (hint: `docker volumes`), manteniendo la convencion de que el cliente N utilizara el archivo de apuestas `.data/agency-{N}.csv` .

En el servidor, si todas las apuestas del *batch* fueron procesadas correctamente, imprimir por log: `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`. En caso de detectar un error con alguna de las apuestas, debe responder con un código de error a elección e imprimir: `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`.

La cantidad máxima de apuestas dentro de cada _batch_ debe ser configurable desde config.yaml. Respetar la clave `batch: maxAmount`, pero modificar el valor por defecto de modo tal que los paquetes no excedan los 8kB.

Por su parte, el servidor deberá responder con éxito solamente si todas las apuestas del _batch_ fueron procesadas correctamente.

### Ejercicio N°7:

Modificar los clientes para que notifiquen al servidor al finalizar con el envío de todas las apuestas y así proceder con el sorteo.
Inmediatamente después de la notificacion, los clientes consultarán la lista de ganadores del sorteo correspondientes a su agencia.
Una vez el cliente obtenga los resultados, deberá imprimir por log: `action: consulta_ganadores | result: success | cant_ganadores: ${CANT}`.

El servidor deberá esperar la notificación de las 5 agencias para considerar que se realizó el sorteo e imprimir por log: `action: sorteo | result: success`.
Luego de este evento, podrá verificar cada apuesta con las funciones `load_bets(...)` y `has_won(...)` y retornar los DNI de los ganadores de la agencia en cuestión. Antes del sorteo no se podrán responder consultas por la lista de ganadores con información parcial.

Las funciones `load_bets(...)` y `has_won(...)` son provistas por la cátedra y no podrán ser modificadas por el alumno.

No es correcto realizar un broadcast de todos los ganadores hacia todas las agencias, se espera que se informen los DNIs ganadores que correspondan a cada una de ellas.

## Parte 3: Repaso de Concurrencia
En este ejercicio es importante considerar los mecanismos de sincronización a utilizar para el correcto funcionamiento de la persistencia.

### Ejercicio N°8:

Modificar el servidor para que permita aceptar conexiones y procesar mensajes en paralelo. En caso de que el alumno implemente el servidor en Python utilizando _multithreading_,  deberán tenerse en cuenta las [limitaciones propias del lenguaje](https://wiki.python.org/moin/GlobalInterpreterLock).

## Condiciones de Entrega
Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
 (hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados  [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).


Le **Bluetooth Low Energy (BLE)** est une technologie de communication sans fil à faible consommation d’énergie, idéale pour les appareils IoT (Internet des objets) comme les capteurs, les dispositifs portables, et bien plus encore. BLE fonctionne par l’échange d’attributs entre un périphérique (généralement un "périphérique" BLE) et un autre périphérique, souvent un téléphone portable ou un ordinateur.

### Concepts de base : 

1. Attributs

Les attributs sont les éléments fondamentaux d'un service BLE. Chaque attribut a un identifiant unique appelé handle, et contient des informations comme une valeur (par exemple, la température mesurée par un capteur) ou des propriétés (comme "lecture" ou "écriture").

2. Services

Un service est une collection d’attributs qui effectuent des tâches spécifiques. Par exemple, un service pourrait être lié à la lecture d’une température, à l’échange d’informations sur la batterie, etc.

    Chaque service est identifié par un UUID (identifiant unique universel).
    Un périphérique BLE peut offrir plusieurs services.

3. Caractéristiques

Les caractéristiques sont des attributs spécifiques à un service qui permettent de lire ou écrire des valeurs. Chaque caractéristique possède un UUID et peut avoir des propriétés (comme "lecture", "écriture", ou "notifications").

    Les caractéristiques peuvent contenir des données de capteurs (température, humidité, etc.).

4. Propriétés des caractéristiques

Les propriétés d'une caractéristique définissent les actions possibles sur celle-ci. Voici les plus courantes :

- 0x02 (Read) : Permet de lire la valeur de la caractéristique.
- 0x04 (Write) : Permet d'écrire une valeur dans la caractéristique sans réponse.
- 0x08 (Write with response) : Permet d’écrire une valeur dans la caractéristique avec une réponse.
- 0x10 (Notify) : Permet au périphérique d'envoyer des notifications lorsque la valeur de la caractéristique change.
- 0x20 (Indicate) : Semblable aux notifications, mais avec un accusé de réception.

### Comment interagir avec un périphérique BLE ?

Pour interagir avec un device BLE plusieurs options s'offrent à vous : 

- Sur linux : gatttool
- Sur Windows : Bluetooth LE explorer, nrfConnect desktop
- En Python avec Bleak
- Sur mobile avec des applications (nrfConnect, BLE scan)
#### Utilisation de gatttool

`gatttool` est un outil en ligne de commande pour interagir avec les périphériques BLE. 
 Il est souvent utilisé dans un environnement Linux pour interagir avec des périphériques BLE via des commandes.

1. Scanner les périphériques BLE :

	`bluetoothctl` -> `power on` -> `scan on`

2. Se connecter à un périphérique :

	`gatttool -b <MAC_ADDRESS> -I` -> `connect`

3. Lister les services du périphérique :

	`primary`

4. Lister les caractéristiques d'un service :

	`characteristics`

5. Lire une caractéristique (attention les valeurs sont en hexa -> convertir en ascii avec xxd, ou bien cyberchef/dcode...) :

	`char-read-hnd <handle>`

6. Écrire une valeur dans une caractéristique (exemple d’écriture en mode sans réponse, idem convertir ascii -> hexa pour écrire) :

    `char-write-cmd <handle> <value>`

Pour les autres options : 

- https://bleak.readthedocs.io/en/latest/index.html
- https://docs.nordicsemi.com/bundle/nrf-connect-desktop/page/index.html

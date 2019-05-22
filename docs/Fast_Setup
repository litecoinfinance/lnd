To use Litecoin ligthning network, you need 

https://github.com/litecoinfinance/lnd - Lightning Network Daemon

https://github.com/litecoinfinance/litecoinfinance - Litecoin Finance full node (core wallet)

Install wallet

Go to data folder

Windows
```
C:\Users\YourUserName\AppData\Roaming\litecoinfinance
```
Linux
```
/home/YourUserName/.litecoinfinance
```
MAC OS 
```
HDD/Users/YourUserName/Library/Application Support/litecoinfinance/ 
```
( if you no see Library folder, at username folder click View > Show View Options > choise Show Library Folder )

Create litecoinfinance.conf and put there this

```
rpcuser=user( make own)
rpcpassword=password (make own)
server=1
listen=1
daemon=1
txindex=1
rpcallowip=127.0.0.1 Allow rpc request from Localhost
rpcport=39327
rpcthreads=8
rpcworkqueue=4096
dbcache=4096
maxmempool=4096
maxorphantx=4096
blockreconstructionextratxn=4096
maxreceivebuffer=4096
maxsendbuffe=4096
datacarrier=4096
datacarriersize=4096
maxconnections=100
zmqpubrawblock=tcp://127.0.0.1:28332
zmqpubrawtx=tcp://127.0.0.1:28333
deprecatedrpc=signrawtransaction
discardfee=0.00000001
mintxfee=0.00000001
minrelaytxfee=0.00000001
```
Now run wallet and wait full sync

Then unpack lnd binary files for your os.

Example

Windows
```
C:/lnd
```
Linux
```
/home/YourUserName/lnd
```
MAC OS
```
HDD/Users/YourUserName/lnd
```
Now we need go to default folder lnd and create conf file at that folder.


Windows go to 
```
C:\Users\YourUserName\AppData\Local\ and create folder lnd
```
Linux go to
```
/home/YourUserName/ and create folder .lnd
```
MAC OS go to
```
HDD/Users/YourUserName/Library/Application Support/ and create folder lnd
```
( if you no see Library folder, at username folder click View > Show View Options > choise Show Library Folder )


No create at that folder lnd.conf

and put there 
```
litecoinfinance.active=1
litecoinfinance.mainnet=1
debuglevel=debug
litecoinfinance.node=litecoinfinanced
litecoinfinanced.rpcuser=user ( RPC user from litecoindfinance wallet)
litecoinfinanced.rpcpass=password ( RPC password from litecoindfinance wallet)
litecoinfinanced.zmqpubrawblock=tcp://127.0.0.1:28332
litecoinfinanced.zmqpubrawtx=tcp://127.0.0.1:28333
externalip=127.0.0.1:9735 ( external ip of your lnd node )
litecoinfinanced.rpchost=127.0.0.1:39327 (ip:port for connect to RPC)
alias=MyWindows_node ( name of your node )
color=#D11711 ( color of node , there is RED)
```

Now open 2 terminals depend at your system and go to folders with lnd and lncli.

Run lnd node

Windows
```
lnd and push enter
```
Linux
```
./lnd and push enter
```
MAC OS
```
./lnd and push enter
```
Now go to another terminal and put

Windows
```
lncli --network mainnet --chain litecoinfinance create
```
Linux
```
./lncli --network mainnet --chain litecoinfinance create
```
MAC OS
```
./lncli --network mainnet --chain litecoinfinance create
```
choise password , bakup seed's and wait full sync node.

Now we need gen new address so put 

Windows
```
lncli --network mainnet --chain litecoinfinance newaddress p2wkh
```
Linux
```
./lncli --network mainnet --chain litecoinfinance newaddress p2wkh
```
MAC OS
```
./lncli --network mainnet --chain litecoinfinance newaddress p2wkh
```
Example

```
C:\lnd>lncli --network mainnet --chain litecoinfinance newaddress p2wkh
{
    "address": "ltfn1qlagrg0t2d9vstdrt7whr5v0xl9pcv2vs5n7nd3"
}
```

and send to that address some coins

After confirmation we can openfirst chanel - for check wallet balance use command 
```
C:\lnd>lncli --network mainnet --chain litecoinfinance walletbalance
```
Example
```
C:\lnd>lncli --network mainnet --chain litecoinfinance walletbalance
{
    "total_balance": "1098998871",
    "confirmed_balance": "1098998871",
    "unconfirmed_balance": "0"
}
```

For open chanel use command 
```
lncli --network mainnet --chain litecoinfinance openchannel 0293795d46bd8b229455ccf1c3de8f290cbb5e4de71a3f60a5b26dab59ca03be34 1000000
```
i was choise random node from peers by 
```
lncli --network mainnet --chain litecoinfinance listpeers
```
```
C:\lnd>lncli --network mainnet --chain litecoinfinance listpeers
{
    "peers": [
        {
            "pub_key": "036fdd52f59fe8a82481d0abd5f9979d2a7fc3c8adb860cd76451ddd2152a9bb39",
            "address": "168.235.67.237:9735",
            "bytes_sent": "3495",
            "bytes_recv": "2782",
            "sat_sent": "0",
            "sat_recv": "0",
            "inbound": false,
            "ping_time": "117687",
            "sync_type": "ACTIVE_SYNC"
        },
        {
            "pub_key": "02ffb92d773a09dd0add352a71c83937d56b0764f561a606fbc3460ad941b5b56e",
            "address": "168.235.67.167:9735",
            "bytes_sent": "3168",
            "bytes_recv": "2544",
            "sat_sent": "0",
            "sat_recv": "0",
            "inbound": false,
            "ping_time": "117646",
            "sync_type": "ACTIVE_SYNC"
        },
        {
            "pub_key": "0293795d46bd8b229455ccf1c3de8f290cbb5e4de71a3f60a5b26dab59ca03be34",
            "address": "107.191.111.59:9735",
            "bytes_sent": "2801",
            "bytes_recv": "2692",
            "sat_sent": "0",
            "sat_recv": "0",
            "inbound": false,
            "ping_time": "128657",
            "sync_type": "ACTIVE_SYNC"
        }
    ]
}
```

Wait 6 confirmation and channel by opened and now you can start create invoce , or pay invoice.
```
C:\lnd>lncli --network mainnet --chain litecoinfinance openchannel 0293795d46bd8b229455ccf1c3de8f290cbb5e4de71a3f60a5b26dab59ca03be34 1000000
{
        "funding_txid": "adc2efef84fba2244fda8d57b24e597ed53cfad1cfc2b98306ddb00e8abb8d52"
}
```

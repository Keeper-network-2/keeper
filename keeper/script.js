const Web3 = require('web3');
const web3 = new Web3('https://eth-sepolia.g.alchemy.com/v2/4-Mm4R00QrwNs-Z_vjOPXjSBfO4m4mmV');

async function encodeFunctionCall(contractAbi, functionName, args) {
    const contract = new web3.eth.Contract(contractAbi);
    const encodedData = contract.methods[functionName](...args).encodeABI();
    return encodedData;
}

const abi = [
    {
        "inputs": [],
        "stateMutability": "nonpayable",
        "type": "constructor"
    },
    {
        "anonymous": false,
        "inputs": [
            {
                "indexed": false,
                "internalType": "uint256",
                "name": "newPrice",
                "type": "uint256"
            }
        ],
        "name": "PriceUpdated",
        "type": "event"
    },
    {
        "inputs": [],
        "name": "owner",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "price",
        "outputs": [
            {
                "internalType": "uint256",
                "name": "",
                "type": "uint256"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "uint256",
                "name": "newPrice",
                "type": "uint256"
            }
        ],
        "name": "updatePrice",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    }
];


const functionName = process.argv[2];
const args = JSON.parse(process.argv[3]);

encodeFunctionCall(abi, functionName, args).then(encodedData => {
    console.log(encodedData);
}).catch(error => {
    console.error('Error encoding function call:', error);
});

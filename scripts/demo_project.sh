#!/usr/bin/env bash

wait() {
  echo "Waiting for chain to start..."
  while :; do
    RET=$(ixocli status 2>&1)
    if [[ ($RET == ERROR*) || ($RET == *'"latest_block_height": "0"'*) ]]; then
      sleep 1
    else
      echo "A few more seconds..."
      sleep 6
      break
    fi
  done
}

RET=$(ixocli status 2>&1)
if [[ ($RET == ERROR*) || ($RET == *'"latest_block_height": "0"'*) ]]; then
  wait
fi

GAS_PRICES="0.025uixo"
PASSWORD="12345678"

PROJECT_DID="did:ixo:U7GK8p8rVhJMKhBVRCJJ8c"
PROJECT_DID_FULL='{
  "did":"did:ixo:U7GK8p8rVhJMKhBVRCJJ8c",
  "verifyKey":"FmwNAfvV2xEqHwszrVJVBR3JgQ8AFCQEVzo1p6x4L8VW",
  "encryptionPublicKey":"domKpTpjrHQtKUnaFLjCuDLe2oHeS4b1sKt7yU9cq7m",
  "secret":{
    "seed":"933e454dbcfc1437f3afc10a0cd512cf0339787b6595819849f53707c268b053",
    "signKey":"Aun1EpjR1HQu1idBsPQ4u4C4dMwtbYPe1SdSC5bUerFC",
    "encryptionPrivateKey":"Aun1EpjR1HQu1idBsPQ4u4C4dMwtbYPe1SdSC5bUerFC"
  }
}'
PROJECT_INFO='{
  "nodeDid":"nodeDid",
  "requiredClaims":"500",
  "serviceEndpoint":"serviceEndpoint",
  "createdOn":"2020-01-01T01:01:01.000Z",
  "createdBy":"Creator",
  "status":"",
  "fees":{
    "@context":"",
    "items": [
      {"@type":"OracleFee", "id":"payment:template:oracle-fee"},
      {"@type":"FeeForService", "id":"payment:template:fee-for-service"}
    ]
  }
}'

ORACLE_FEE_PAYMENT_TEMPLATE='{
  "id": "payment:template:oracle-fee",
  "payment_amount": [{"denom":"uixo", "amount":"5000000"}],
  "payment_minimum": [{"denom":"uixo", "amount":"5000000"}],
  "payment_maximum": [{"denom":"uixo", "amount":"50000000"}],
  "discounts": []
}'
FEE_FOR_SERVICE_PAYMENT_TEMPLATE='{
  "id": "payment:template:fee-for-service",
  "payment_amount": [{"denom":"uixo", "amount":"2000000"}],
  "payment_minimum": [{"denom":"uixo", "amount":"2000000"}],
  "payment_maximum": [{"denom":"uixo", "amount":"20000000"}],
  "discounts": []
}'

MIGUEL_DID="did:ixo:4XJLBfGtWSGKSz4BeRxdun"
FRANCESCO_DID="did:ixo:UKzkhVSHc3qEFva5EY2XHt"
SHAUN_DID="did:ixo:U4tSpzzv91HHqWW1YmFkHJ"
MIGUEL_DID_FULL='{
  "did":"did:ixo:4XJLBfGtWSGKSz4BeRxdun",
  "verifyKey":"2vMHhssdhrBCRFiq9vj7TxGYDybW4yYdrYh9JG56RaAt",
  "encryptionPublicKey":"6GBp8qYgjE3ducksUa9Ar26ganhDFcmYfbZE9ezFx5xS",
  "secret":{
    "seed":"38734eeb53b5d69177da1fa9a093f10d218b3e0f81087226be6ce0cdce478180",
    "signKey":"4oMozrMR6BXRN93MDk6UYoqBVBLiPn9RnZhR3wQd6tBh",
    "encryptionPrivateKey":"4oMozrMR6BXRN93MDk6UYoqBVBLiPn9RnZhR3wQd6tBh"
  }
}'
FRANCESCO_DID_FULL='{
  "did":"did:ixo:UKzkhVSHc3qEFva5EY2XHt",
  "verifyKey":"Ftsqjc2pEvGLqBtgvVx69VXLe1dj2mFzoi4kqQNGo3Ej",
  "encryptionPublicKey":"8YScf3mY4eeHoxDT9MRxiuGX5Fw7edWFnwHpgWYSn1si",
  "secret":{
    "seed":"94f3c48a9b19b4881e582ba80f5767cd3f3c5d7b7103cb9a50fa018f108d89de",
    "signKey":"B2Svs8GoQnUJHg8W2Ch7J53Goq36AaF6C6W4PD2MCPrM",
    "encryptionPrivateKey":"B2Svs8GoQnUJHg8W2Ch7J53Goq36AaF6C6W4PD2MCPrM"
  }
}'
SHAUN_DID_FULL='{
  "did":"did:ixo:U4tSpzzv91HHqWW1YmFkHJ",
  "verifyKey":"FkeDue5it82taeheMprdaPrctfK3DeVV9NnEPYDgwwRG",
  "encryptionPublicKey":"DtdGbZB2nSQvwhs6QoN5Cd8JTxWgfVRAGVKfxj8LA15i",
  "secret":{
    "seed":"6ef0002659d260a0bbad194d1aa28650ccea6c6862f994dfdbd48648e1a05c5e",
    "signKey":"8U474VrG2QiUFKfeNnS84CAsqHdmVRjEx4vQje122ycR",
    "encryptionPrivateKey":"8U474VrG2QiUFKfeNnS84CAsqHdmVRjEx4vQje122ycR"
  }
}'

# Ledger DIDs
echo "Ledgering Miguel DID..."
ixocli tx did add-did-doc "$MIGUEL_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Ledgering Francesco DID..."
ixocli tx did add-did-doc "$FRANCESCO_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Ledgering Shaun DID..."
ixocli tx did add-did-doc "$SHAUN_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# Create oracle fee payment template
echo "Creating oracle fee payment template..."
CREATOR="$FRANCESCO_DID_FULL"
ixocli tx payments create-payment-template "$ORACLE_FEE_PAYMENT_TEMPLATE" "$CREATOR" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# Create fee-for-service payment template
echo "Creating fee-for-service payment template..."
CREATOR="$FRANCESCO_DID_FULL"
ixocli tx payments create-payment-template "$FEE_FOR_SERVICE_PAYMENT_TEMPLATE" "$CREATOR" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# Create project and progress status to PENDING
SENDER_DID="$SHAUN_DID"
echo "Creating project..."
ixocli tx project create-project "$SENDER_DID" "$PROJECT_INFO" "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Updating project to CREATED..."
ixocli tx project update-project-status "$SENDER_DID" CREATED "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Updating project to PENDING..."
ixocli tx project update-project-status "$SENDER_DID" PENDING "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# Fund project and progress status to FUNDED
PROJECT_ADDR=$(ixocli q project get-project-accounts $PROJECT_DID | grep "$PROJECT_DID" | cut -d \" -f 4)
echo "Funding project at [$PROJECT_ADDR] with uixo (using treasury 'oracle-transfer' using Francesco oracle)..."
ixocli tx treasury oracle-mint "$PROJECT_ADDR" 100000000uixo "$FRANCESCO_DID_FULL" "dummy proof" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Updating project to FUNDED..."
SENDER_DID="$SHAUN_DID"
ixocli tx project update-project-status "$SENDER_DID" FUNDED "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# Progress project status to STARTED
SENDER_DID="$SHAUN_DID"
echo "Updating project to STARTED..."
ixocli tx project update-project-status "$SENDER_DID" STARTED "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# Create claim and evaluation
echo "Creating a claim in project..."
SENDER_DID="$SHAUN_DID"
ixocli tx project create-claim "tx_hash" "$SENDER_DID" "claim_id" "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Creating an evaluation in project..."
SENDER_DID="$MIGUEL_DID"
STATUS="1" # create-evaluation updates status of claim from 0 to 1
ixocli tx project create-evaluation "tx_hash" "$SENDER_DID" "claim_id" $STATUS "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y

# OracleFeePercentage:  0.1 (10%)
# NodeFeePercentage:    0.1 (10%)

# Fee for service:   2000000 uixo
# Oracle pay:        5000000 uixo

# Expected project account balances:
# - InitiatingNodePayFees:    50000  # 0.1 of 0.1 of oracle pay
# - IxoFees:                      0
# - IxoPayFees:              450000  # 0.9 of 0.1 of oracle pay
# - project:               93000000  # 100IXO - (5+2)IXO
# Expected external account balances:
# - Miguel:                 4500000  # 0.9 of oracle pay
# - Shaun:                 11000000  # 1.0 of fee-for-service (+ 10000000 original - 1000000 project creation fee)

# Sum of fee accounts: 500000

# Progress project status to PAIDOUT
SENDER_DID="$SHAUN_DID"
echo "Updating project to STOPPED..."
ixocli tx project update-project-status "$SENDER_DID" STOPPED "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Updating project to PAIDOUT..."
ixocli tx project update-project-status "$SENDER_DID" PAIDOUT "$PROJECT_DID_FULL" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Project withdrawals query..."
ixocli q project get-project-txs $PROJECT_DID

# Expected withdrawals:
# - 500000 to ixo (a.k.a Shaun) DID (did:ixo:U4tSpzzv91HHqWW1YmFkHJ)
# Expected project account balances:
# - InitiatingNodePayFees:        0
# - IxoFees:                      0
# - IxoPayFees:                   0
# - project:               93000000
# Expected external account balances:
# - Miguel:                 4500000
# - Shaun:                 11500000  # 500000 withdrawal

echo "InitiatingNodePayFees"
ixocli q auth account "ixo1xvjy68xrrtxnypwev9r8tmjys9wk0zkkspzjmq"
echo "IxoFees"
ixocli q auth account "ixo1ff9we62w6eyes7wscjup3p40vy4uz0sa7j0ajc"
echo "IxoPayFees"
ixocli q auth account "ixo1udgxtf6yd09mwnnd0ljpmeq4vnyhxdg03uvne3"
echo "did:ixo:U7GK8p8rVhJMKhBVRCJJ8c"
ixocli q auth account "ixo1rmkak6t606wczsps9ytpga3z4nre4z3nwc04p8"
echo "did:ixo:4XJLBfGtWSGKSz4BeRxdun"
ixocli q auth account "$(ixocli q did get-address-from-did $MIGUEL_DID)"
echo "did:ixo:U4tSpzzv91HHqWW1YmFkHJ"
ixocli q auth account "$(ixocli q did get-address-from-did $SHAUN_DID)"

# Withdraw funds (from main project account, i.e. as refund)
# --> FAIL since Miguel is not the project owner
echo "Withdraw project funds as Miguel (fail since Miguel is not the owner)..."
DATA="{\"projectDid\":\"$PROJECT_DID\",\"recipientDid\":\"$MIGUEL_DID\",\"amount\":\"100000000\",\"isRefund\":true}"
ixocli tx project withdraw-funds "$MIGUEL_DID_FULL" "$DATA" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Project withdrawals query..."
ixocli q project get-project-txs $PROJECT_DID

# Expected external account balances:
# - Miguel:                 4495000 (5000uixo tx fee deducted)

# Withdraw funds (from main project account, i.e. as refund)
# --> SUCCESS since Shaun is the project owner
echo "Withdraw project funds as Shaun (success since Shaun is the owner)..."
DATA="{\"projectDid\":\"$PROJECT_DID\",\"recipientDid\":\"$SHAUN_DID\",\"amount\":\"1000000\",\"isRefund\":true}"
ixocli tx project withdraw-funds "$SHAUN_DID_FULL" "$DATA" --broadcast-mode block --gas-prices="$GAS_PRICES" -y
echo "Project withdrawals query..."
ixocli q project get-project-txs $PROJECT_DID

# Expected withdrawals:
# - 500000 to ixo (a.k.a Shaun) DID (did:ixo:U4tSpzzv91HHqWW1YmFkHJ)
# - 1000000 to shaun DID (did:ixo:U4tSpzzv91HHqWW1YmFkHJ)
# Expected project account balances:
# - InitiatingNodePayFees:        0
# - IxoFees:                      0
# - IxoPayFees:                   0
# - project:               92000000  # 1000000 has been withdrawn
# Expected external account balances:
# - Miguel:                 4495000
# - Shaun:                 12495000  # 1000000 withdrawal + 5000 fee deducted

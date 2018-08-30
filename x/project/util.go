package project

import (
	"github.com/ixofoundation/ixo-cosmos/x/ixo/sovrin"
)

func NewCreateProjectMsg(projectDoc ProjectDoc, projectDid sovrin.SovrinDid) CreateProjectMsg {
	return CreateProjectMsg{
		ProjectMsg: ProjectMsg{
			Original:   "",
			TxHash:     "",
			SenderDid:  "",
			ProjectDid: projectDid.Did,
		},
		PubKey: projectDid.VerifyKey,
		Data:   projectDoc,
	}
}

func NewCreateAgentMsg(txHash string, senderDid string, createAgentDoc CreateAgentDoc, projectDid sovrin.SovrinDid) CreateAgentMsg {
	return CreateAgentMsg{
		ProjectMsg: ProjectMsg{
			Original:   "",
			ProjectDid: projectDid.Did,
			TxHash:     txHash,
			SenderDid:  senderDid,
		},
		Data: createAgentDoc,
	}
}

func NewUpdateAgentMsg(txHash string, senderDid string, updateAgentDoc UpdateAgentDoc, projectDid sovrin.SovrinDid) UpdateAgentMsg {
	return UpdateAgentMsg{
		ProjectMsg: ProjectMsg{
			Original:   "",
			ProjectDid: projectDid.Did,
			TxHash:     txHash,
			SenderDid:  senderDid,
		},
		Data: updateAgentDoc,
	}
}

func NewCreateClaimMsg(txHash string, senderDid string, createClaimDoc CreateClaimDoc, projectDid sovrin.SovrinDid) CreateClaimMsg {
	return CreateClaimMsg{
		ProjectMsg: ProjectMsg{
			Original:   "",
			ProjectDid: projectDid.Did,
			TxHash:     txHash,
			SenderDid:  senderDid,
		},
		Data: createClaimDoc,
	}
}

func NewCreateEvaluationMsg(txHash string, senderDid string, createEvaluationDoc CreateEvaluationDoc, projectDid sovrin.SovrinDid) CreateEvaluationMsg {
	return CreateEvaluationMsg{
		ProjectMsg: ProjectMsg{
			Original:   "",
			ProjectDid: projectDid.Did,
			TxHash:     txHash,
			SenderDid:  senderDid,
		},
		Data: createEvaluationDoc,
	}
}

func NewFundProjectMsg(txHash string, senderDid string, fundProjectDoc FundProjectDoc, projectDid sovrin.SovrinDid) FundProjectMsg {
	// This did is initialized when the node is initialized
	fundProjectDoc.Signer = "ETH_PEG"
	return FundProjectMsg{
		ProjectMsg: ProjectMsg{
			Original:   "",
			ProjectDid: projectDid.Did,
			TxHash:     txHash,
			SenderDid:  senderDid,
		},
		Data: fundProjectDoc,
	}
}

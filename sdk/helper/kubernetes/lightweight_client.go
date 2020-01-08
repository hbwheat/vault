package kubernetes

func NewLightWeightClient() LightWeightClient {
	return &lightWeightClient{}
}

// TODO audit all methods called on the client and
// add them to the interface here, then swap them out.
type LightWeightClient interface{}

type lightWeightClient struct{}

package dacidentity

import (
	"encoding/json"
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
)

type DacConfig struct {
	Hbytes      []byte   `json:"h"`
	YsBytes1    [][]byte `json:"ys1"`
	YsBytes2    [][]byte `json:"ys2"`
	RootPkBytes []byte   `json:"rootpk"`
}

func (c *DacConfig) H() (interface{}, error) {
	return dac.PointFromBytes(c.Hbytes)
}

func (c *DacConfig) Ys() ([][]interface{}, error) {
	var err error
	Ys := make([][]interface{}, 2)
	Ys[0] = make([]interface{}, len(c.YsBytes2))
	for index, yBytes := range c.YsBytes2 {
		Ys[0][index], err = dac.PointFromBytes(yBytes)
		if err != nil {
			return nil, err
		}
	}
	Ys[1] = make([]interface{}, len(c.YsBytes1))
	for index, yBytes := range c.YsBytes1 {
		Ys[1][index], err = dac.PointFromBytes(yBytes)
		if err != nil {
			return nil, err
		}
	}
	return Ys, nil
}

func (c *DacConfig) RootPk() (interface{}, error) {
	return dac.PointFromBytes(c.RootPkBytes)
}

func CreateConfig() (*DacConfig, dac.SK) {
	YsNum := 10
	prg := NewRand()
	dacConfig := DacConfig{}

	h := FP256BN.ECP2_generator().Mul(FP256BN.Randomnum(FP256BN.NewBIGints(FP256BN.CURVE_Order), prg))
	dacConfig.Hbytes = dac.PointToBytes(h)

	dacConfig.YsBytes1 = make([][]byte, YsNum)
	for index, y := range dac.GenerateYs(true, YsNum, prg) {
		dacConfig.YsBytes1[index] = dac.PointToBytes(y)
	}
	dacConfig.YsBytes2 = make([][]byte, YsNum)
	for index, y := range dac.GenerateYs(false, YsNum, prg) {
		dacConfig.YsBytes2[index] = dac.PointToBytes(y)
	}

	RootSk, RootPk := dac.GenerateKeys(prg, 0)
	dacConfig.RootPkBytes = dac.PointToBytes(RootPk)

	return &dacConfig, RootSk
}

func CreateConfigFromBytes(configBytes []byte) (*DacConfig, error) {
	dacConfig := DacConfig{}

	err := json.Unmarshal(configBytes, &dacConfig)
	if err != nil {
		return nil, err
	}

	return &dacConfig, nil
}

type CredentialsConfig struct {
	CredentialsBytes []byte `json:"credentials"`
	PkBytes []byte `json:"pk"`
	SkBytes []byte `json:"sk"`
}

func (credsConfig *CredentialsConfig) Credentials() *dac.Credentials {
	return dac.CredentialsFromBytes(credsConfig.CredentialsBytes)
}

func (credsConfig *CredentialsConfig) Pk() (dac.PK, error) {
	return dac.PointFromBytes(credsConfig.PkBytes)
}

func (credsConfig *CredentialsConfig) Sk() dac.SK {
	return FP256BN.FromBytes(credsConfig.SkBytes)
}
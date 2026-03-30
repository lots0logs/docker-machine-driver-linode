package linode

import (
	"net"
	"reflect"
	"testing"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestSetConfigFromFlags(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"linode-token":     "PROJECT",
			"linode-root-pass": "ROOTPASS",
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)

	assert.NoError(t, err)
	assert.Empty(t, checkFlags.InvalidFlags)
}

func TestSetConfigFromFlagsInterfaceRequiresVPC(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"linode-token":          "PROJECT",
			"linode-use-interfaces": true,
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires either existing VPC IDs or")
}

func TestSetConfigFromFlagsInterfaceConflictsWithLegacyPrivateIP(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"linode-token":             "PROJECT",
			"linode-use-interfaces":    true,
			"linode-vpc-id":            123,
			"linode-vpc-subnet-id":     456,
			"linode-create-private-ip": true,
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "linode-use-interfaces")
}

func TestSetConfigFromFlagsInterfaceHappyPath(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"linode-token":          "PROJECT",
			"linode-use-interfaces": true,
			"linode-vpc-id":         123,
			"linode-vpc-subnet-id":  456,
			"linode-vpc-private-ip": "10.0.0.10",
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)
	assert.NoError(t, err)
	assert.True(t, driver.UseInterfaces)
	assert.Equal(t, 123, driver.VPCID)
	assert.Equal(t, 456, driver.VPCSubnetID)
	assert.Equal(t, "10.0.0.10", driver.VPCPrivateIP)
}

func TestSetConfigFromFlagsInterfaceCreateNewVPC(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"linode-token":             "PROJECT",
			"linode-use-interfaces":    true,
			"linode-vpc-label":         "new-vpc",
			"linode-vpc-subnet-label":  "subnet-a",
			"linode-vpc-subnet-ipv4":   "10.0.0.0/24",
			"linode-vpc-private-ip":    "10.0.0.10",
			"linode-create-private-ip": false,
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)
	assert.NoError(t, err)
	assert.True(t, driver.UseInterfaces)
	assert.Zero(t, driver.VPCID)
	assert.Zero(t, driver.VPCSubnetID)
	assert.Equal(t, "new-vpc", driver.VPCLabel)
	assert.Equal(t, "subnet-a", driver.VPCSubnetLabel)
	assert.Equal(t, "10.0.0.0/24", driver.VPCSubnetIPv4)
	assert.Equal(t, "10.0.0.10", driver.VPCPrivateIP)
}

func TestSetConfigFromFlagsInterfaceMixedVPCInputs(t *testing.T) {
	driver := NewDriver("", "")

	checkFlags := &drivers.CheckDriverOptions{
		FlagsValues: map[string]interface{}{
			"linode-token":            "PROJECT",
			"linode-use-interfaces":   true,
			"linode-vpc-id":           123,
			"linode-vpc-subnet-label": "subnet-a",
		},
		CreateFlags: driver.GetCreateFlags(),
	}

	err := driver.SetConfigFromFlags(checkFlags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires both --linode-vpc-id and --linode-vpc-subnet-id")
}

func TestPrivateIP(t *testing.T) {
	ip := net.IP{}
	for _, addr := range [][]byte{
		[]byte("172.16.0.1"),
		[]byte("192.168.0.1"),
		[]byte("10.0.0.1"),
	} {
		if err := ip.UnmarshalText(addr); err != nil {
			t.Error(err)
		}
		assert.True(t, privateIP(ip))
	}

	if err := ip.UnmarshalText([]byte("1.1.1.1")); err != nil {
		t.Error(err)
	}
	assert.False(t, privateIP(ip))
}

func TestIPInCIDR(t *testing.T) {
	tenOne := net.IP{}

	if err := tenOne.UnmarshalText([]byte("10.0.0.1")); err != nil {
		t.Error(err)
	}
	assert.True(t, ipInCIDR(tenOne, "10.0.0.0/8"), "10.0.0.1 is in 10.0.0.0/8")
	assert.False(t, ipInCIDR(tenOne, "254.0.0.0/8"), "10.0.0.1 is not in 254.0.0.0/8")
}

func TestNormalizeInstanceLabel(t *testing.T) {
	inputLabel := "_mycoollabel25';./__----=][[this,label,is,really[good]and]long[wow+that'scrazy[]what[a\\good!labelname."
	expectedResult := "mycoollabel25._-thislabelisreallygoodandlongwowthatscrazywhatago"

	result, err := normalizeInstanceLabel(inputLabel)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatal(cmp.Diff(result, expectedResult))
	}
}

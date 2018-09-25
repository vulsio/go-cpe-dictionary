package testutil

import (
	"github.com/alicebob/miniredis"
)

// PrepareTestRedis prepares redis for test
func PrepareTestRedis() (s *miniredis.Miniredis, err error) {
	s, err = miniredis.Run()
	if err != nil {
		return nil, err
	}

	s.ZAdd("CPE#VendorProduct", 0, "vendorName1::productName1-1")
	s.ZAdd("CPE#VendorProduct", 0, "vendorName1::productName1-2")
	s.ZAdd("CPE#VendorProduct", 0, "vendorName2::productName2")
	s.ZAdd("CPE#VendorProduct", 0, "vendorName3::productName3")
	s.ZAdd("CPE#VendorProduct", 0, "vendorName4::productName4")
	s.ZAdd("CPE#VendorProduct", 0, "vendorName5::productName5")
	s.ZAdd("CPE#VendorProduct", 0, "vendorName6::productName6")

	s.ZAdd("CPE#CpeURI", 0, "cpe:/a:vendorName1:productName1-1:1.1")
	s.ZAdd("CPE#CpeURI", 0, "cpe:/a:vendorName1:productName1-2:1.2")
	s.ZAdd("CPE#CpeURI", 0, "cpe:/o:vendorName2:productName2:2.0")
	s.ZAdd("CPE#CpeURI", 0, "cpe:/h:vendorName3:productName3:3.0")
	s.ZAdd("CPE#CpeURI", 0, "cpe:/a:vendorName4:productName4:4.0")
	s.ZAdd("CPE#CpeURI", 0, "cpe:/o:vendorName5:productName5:5.0")
	s.ZAdd("CPE#CpeURI", 0, "cpe:/h:vendorName6:productName6:6.0")

	s.ZAdd("CPE#vendorName1::productName1-1", 0, "cpe:/a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*")
	s.ZAdd("CPE#vendorName1::productName1-2", 0, "cpe:/a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*")
	s.ZAdd("CPE#vendorName2::productName2", 0, "cpe:/a:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*")
	s.ZAdd("CPE#vendorName3::productName3", 0, "cpe:/a:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*")
	s.ZAdd("CPE#vendorName4::productName4", 0, "cpe:/a:vendorName4:productName4:4.0:*:*:*:*:targetSoftware4:targetHardware4:*")
	s.ZAdd("CPE#vendorName5::productName5", 0, "cpe:/a:vendorName5:productName5:5.0:*:*:*:*:targetSoftware5:targetHardware5:*")
	s.ZAdd("CPE#vendorName6::productName6", 0, "cpe:/a:vendorName6:productName6:6.0:*:*:*:*:targetSoftware6:targetHardware6:*")
	return s, nil
}

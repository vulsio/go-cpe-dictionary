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

	s.ZAdd("CPE#Vendor", 0, "vendorName1")
	s.ZAdd("CPE#Vendor", 0, "vendorName2")
	s.ZAdd("CPE#Vendor", 0, "vendorName3")
	s.ZAdd("CPE#Vendor", 0, "vendorName4")
	s.ZAdd("CPE#Vendor", 0, "vendorName5")
	s.ZAdd("CPE#Vendor", 0, "vendorName6")
	s.ZAdd("CPE#Product", 0, "productName1-1")
	s.ZAdd("CPE#Product", 0, "productName1-2")
	s.ZAdd("CPE#Product", 0, "productName2")
	s.ZAdd("CPE#Product", 0, "productName3")
	s.ZAdd("CPE#Product", 0, "productName4")
	s.ZAdd("CPE#Product", 0, "productName5")
	s.ZAdd("CPE#Product", 0, "productName6")
	s.ZAdd("CPE#VendorProduct::vendorName1", 0, "productName1-1")
	s.ZAdd("CPE#VendorProduct::vendorName1", 0, "productName1-2")
	s.ZAdd("CPE#VendorProduct::vendorName2", 0, "productName2")
	s.ZAdd("CPE#VendorProduct::vendorName3", 0, "productName3")
	s.ZAdd("CPE#VendorProduct::vendorName4", 0, "productName4")
	s.ZAdd("CPE#VendorProduct::vendorName5", 0, "productName5")
	s.ZAdd("CPE#VendorProduct::vendorName6", 0, "productName6")
	s.ZAdd("CPE#TargetSoftware", 0, "targetSoftware1")
	s.ZAdd("CPE#TargetSoftware", 0, "targetSoftware2")
	s.ZAdd("CPE#TargetSoftware", 0, "targetSoftware3")
	s.ZAdd("CPE#TargetSoftware", 0, "targetSoftware4")
	s.ZAdd("CPE#TargetSoftware", 0, "targetSoftware5")
	s.ZAdd("CPE#TargetSoftware", 0, "targetSoftware6")
	s.ZAdd("CPE#TargetHardware", 0, "targetHardware1")
	s.ZAdd("CPE#TargetHardware", 0, "targetHardware2")
	s.ZAdd("CPE#TargetHardware", 0, "targetHardware3")
	s.ZAdd("CPE#TargetHardware", 0, "targetHardware4")
	s.ZAdd("CPE#TargetHardware", 0, "targetHardware5")
	s.ZAdd("CPE#TargetHardware", 0, "targetHardware6")

	s.HSet("CPE#Cpe22", "cpe:/a:vendorName1:productName1-1:1.1", cpe11)
	s.HSet("CPE#Cpe22", "cpe:/a:vendorName1:productName1-2:1.2", cpe12)
	s.HSet("CPE#Cpe22", "cpe:/o:vendorName2:productName2:2.0", cpe2)
	s.HSet("CPE#Cpe22", "cpe:/h:vendorName3:productName3:3.0", cpe3)
	s.HSet("CPE#Cpe22", "cpe:/a:vendorName4:productName4:4.0", cpe4)
	s.HSet("CPE#Cpe22", "cpe:/o:vendorName5:productName5:5.0", cpe5)
	s.HSet("CPE#Cpe22", "cpe:/h:vendorName6:productName6:6.0", cpe6)

	s.HSet("CPE#Cpe23", "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*", cpe11)
	s.HSet("CPE#Cpe23", "cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*", cpe12)
	s.HSet("CPE#Cpe23", "cpe:2.3:o:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*", cpe2)
	s.HSet("CPE#Cpe23", "cpe:2.3:h:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*", cpe3)
	s.HSet("CPE#Cpe23", "cpe:2.3:a:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*", cpe4)
	s.HSet("CPE#Cpe23", "cpe:2.3:o:vendorName4:productName4:4.0:*:*:*:*:targetSoftware4:targetHardware4:*", cpe5)
	s.HSet("CPE#Cpe23", "cpe:2.3:h:vendorName5:productName5:5.0:*:*:*:*:targetSoftware5:targetHardware5:*", cpe6)

	s.HSet("CPE#VendorProduct::vendorName1::productName1-1", "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*", cpe11)
	s.HSet("CPE#VendorProduct::vendorName1::productName1-2", "cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*", cpe12)
	s.HSet("CPE#VendorProduct::vendorName2::productName2", "cpe:2.3:a:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*", cpe2)
	s.HSet("CPE#VendorProduct::vendorName3::productName3", "cpe:2.3:a:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*", cpe3)
	s.HSet("CPE#VendorProduct::vendorName4::productName4", "cpe:2.3:a:vendorName4:productName4:4.0:*:*:*:*:targetSoftware4:targetHardware4:*", cpe4)
	s.HSet("CPE#VendorProduct::vendorName5::productName5", "cpe:2.3:a:vendorName5:productName5:5.0:*:*:*:*:targetSoftware5:targetHardware5:*", cpe5)
	s.HSet("CPE#VendorProduct::vendorName6::productName6", "cpe:2.3:a:vendorName6:productName6:6.0:*:*:*:*:targetSoftware6:targetHardware6:*", cpe6)
	return s, nil
}

var (
	cpe11 = `{
    	"Cpe22URI": "cpe:/a:vendorName1:productName1-1:1.1",
    	"Cpe23URI": "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
    	"Part": "a",
    	"Vendor": "vendorName1",
    	"Product": "productName1-1",
    	"Version": "1.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware1",
    	"TargetHardware": "targetHardware1",
    	"Other": "*"
    }`
	cpe12 = `{
    	"Cpe22URI": "cpe:/a:vendorName1:productName1-2:1.2",
    	"Cpe23URI": "cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*",
    	"Part": "a",
    	"Vendor": "vendorName1",
    	"Product": "productName1-2",
    	"Version": "1.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware1",
    	"TargetHardware": "targetHardware1",
    	"Other": "*"
    }`
	cpe2 = `{
    	"Cpe22URI": "cpe:/o:vendorName2:productName2:2.0",
    	"Cpe23URI": "cpe:2.3:o:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*",
    	"Part": "o",
    	"Vendor": "vendorName2",
    	"Product": "productName2",
    	"Version": "2.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware2",
    	"TargetHardware": "targetHardware2",
    	"Other": "*"
    }`
	cpe3 = `{
    	"Cpe22URI": "cpe:/h:vendorName3:productName3:3.0",
    	"Cpe23URI": "cpe:2.3:h:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*",
    	"Part": "h",
    	"Vendor": "vendorName3",
    	"Product": "productName3",
    	"Version": "3.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware3",
    	"TargetHardware": "targetHardware3",
    	"Other": "*"
    }`
	cpe4 = `{
    	"Cpe22URI": "cpe:/a:vendorName4:productName4:4.0",
    	"Cpe23URI": "cpe:2.3:a:vendorName4:productName4:4.0:*:*:*:*:targetSoftware4:targetHardware4:*",
    	"Part": "a",
    	"Vendor": "vendorName4",
    	"Product": "productName4",
    	"Version": "4.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware4",
    	"TargetHardware": "targetHardware4",
    	"Other": "*"
    }`
	cpe5 = `{
    	"Cpe22URI": "cpe:/h:vendorName5:productName5:5.0",
    	"Cpe23URI": "cpe:2.3:h:vendorName5:productName5:5.0:*:*:*:*:targetSoftware5:targetHardware5:*",
    	"Part": "h",
    	"Vendor": "vendorName5",
    	"Product": "productName5",
    	"Version": "5.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware5",
    	"TargetHardware": "targetHardware5",
    	"Other": "*"
    }`
	cpe6 = `{
    	"Cpe22URI": "cpe:/o:vendorName6:productName6:6.0",
    	"Cpe23URI": "cpe:2.3:o:vendorName6:productName6:6.0:*:*:*:*:targetSoftware6:targetHardware6:*",
    	"Part": "o",
    	"Vendor": "vendorName6",
    	"Product": "productName6",
    	"Version": "5.0",
    	"Update": "*",
    	"Edition": "*",
    	"Language": "*",
    	"SoftwareEdition": "*",
    	"TargetSoftware": "targetSoftware6",
    	"TargetHardware": "targetHardware6",
    	"Other": "*"
    }`
)

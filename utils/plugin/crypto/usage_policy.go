package sealcrypto

func splitRSAUsages(algorithm string, usages []string) ([]string, []string) {
	var pubUsages []string
	var priUsages []string
	pubAllowed := map[string]struct{}{}
	priAllowed := map[string]struct{}{}
	switch algorithm {
	case "RSASSA-PKCS1-V1_5", "RSA-PSS":
		pubAllowed["verify"] = struct{}{}
		priAllowed["sign"] = struct{}{}
	case "RSA-OAEP", "RSAES-PKCS1-V1_5":
		pubAllowed["encrypt"] = struct{}{}
		pubAllowed["wrapKey"] = struct{}{}
		priAllowed["decrypt"] = struct{}{}
		priAllowed["unwrapKey"] = struct{}{}
	default:
		return nil, usages
	}

	for _, u := range usages {
		if _, ok := pubAllowed[u]; ok {
			pubUsages = append(pubUsages, u)
		}
		if _, ok := priAllowed[u]; ok {
			priUsages = append(priUsages, u)
		}
	}

	if len(pubUsages) == 0 {
		for u := range pubAllowed {
			pubUsages = append(pubUsages, u)
		}
	}
	if len(priUsages) == 0 {
		for u := range priAllowed {
			priUsages = append(priUsages, u)
		}
	}
	return pubUsages, priUsages
}

func splitECUsages(algorithm string, usages []string) ([]string, []string) {
	var pubUsages []string
	var priUsages []string
	pubAllowed := map[string]struct{}{}
	priAllowed := map[string]struct{}{}
	switch algorithm {
	case "ECDSA":
		pubAllowed["verify"] = struct{}{}
		priAllowed["sign"] = struct{}{}
	case "ECDH":
		pubAllowed["deriveBits"] = struct{}{}
		pubAllowed["deriveKey"] = struct{}{}
		priAllowed["deriveBits"] = struct{}{}
		priAllowed["deriveKey"] = struct{}{}
	default:
		return nil, usages
	}

	for _, u := range usages {
		if _, ok := pubAllowed[u]; ok {
			pubUsages = append(pubUsages, u)
		}
		if _, ok := priAllowed[u]; ok {
			priUsages = append(priUsages, u)
		}
	}
	if len(pubUsages) == 0 {
		for u := range pubAllowed {
			pubUsages = append(pubUsages, u)
		}
	}
	if len(priUsages) == 0 {
		for u := range priAllowed {
			priUsages = append(priUsages, u)
		}
	}
	return pubUsages, priUsages
}

func splitOKPUsages(algorithm string, usages []string) ([]string, []string) {
	var pubUsages []string
	var priUsages []string
	pubAllowed := map[string]struct{}{}
	priAllowed := map[string]struct{}{}
	switch algorithm {
	case "Ed25519":
		pubAllowed["verify"] = struct{}{}
		priAllowed["sign"] = struct{}{}
	case "X25519":
		pubAllowed["deriveBits"] = struct{}{}
		pubAllowed["deriveKey"] = struct{}{}
		priAllowed["deriveBits"] = struct{}{}
		priAllowed["deriveKey"] = struct{}{}
	default:
		return nil, usages
	}
	for _, u := range usages {
		if _, ok := pubAllowed[u]; ok {
			pubUsages = append(pubUsages, u)
		}
		if _, ok := priAllowed[u]; ok {
			priUsages = append(priUsages, u)
		}
	}
	if len(pubUsages) == 0 {
		for u := range pubAllowed {
			pubUsages = append(pubUsages, u)
		}
	}
	if len(priUsages) == 0 {
		for u := range priAllowed {
			priUsages = append(priUsages, u)
		}
	}
	return pubUsages, priUsages
}

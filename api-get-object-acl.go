package minio

import (
	"context"
	"net/http"
	"net/url"
)

type accessControlPolicy struct {
	Owner struct {
		ID          string `xml:"ID"`
		DisplayName string `xml:"DisplayName"`
	} `xml:"Owner"`
	AccessControlList struct {
		Grant []struct {
			Grantee struct {
				ID          string `xml:"ID"`
				DisplayName string `xml:"DisplayName"`
				XmlnsXsi    string `xml:"_xmlns:xsi"`
				XsiType     string `xml:"_xsi:type"`
				URI         string `xml:"URI"`
			} `xml:"Grantee"`
			Permission string `xml:"Permission"`
		} `xml:"Grant"`
	} `xml:"AccessControlList"`
}

//GetObjectACLS get object ACLs
func (c Client) GetObjectACLS(bucketName, objectName string) (*ObjectInfo, error) {

	resp, err := c.executeMethod(context.Background(), "GET", requestMetadata{
		bucketName: bucketName,
		objectName: objectName,
		queryValues: url.Values{
			"acl": []string{""},
		},
	})
	if err != nil {
		return nil, err
	}
	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp, bucketName, objectName)
	}

	res := &accessControlPolicy{}

	defer resp.Body.Close()
	if err := xmlDecoder(resp.Body, res); err != nil {
		return nil, err
	}

	objInfo, err := c.statObject(context.Background(), bucketName, objectName, StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	cannedACL := getCannedACL(res)
	if cannedACL != "" {
		objInfo.Metadata.Add("x-amz-acl", cannedACL)
		return &objInfo, nil
	}

	grantACL := getAmzGrantACL(res)
	for k, v := range grantACL {
		objInfo.Metadata[k] = v
	}

	return &objInfo, nil
}

func getCannedACL(aCPolicy *accessControlPolicy) string {
	grants := aCPolicy.AccessControlList.Grant

	switch {
	case len(grants) == 1:
		if grants[0].Grantee.URI == "" && grants[0].Permission == "FULL_CONTROL" {
			return "private"
		}
	case len(grants) == 2:
		for _, g := range grants {
			if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AuthenticatedUsers" && g.Permission == "READ" {
				return "authenticated-read"
			}
			if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == "READ" {
				return "public-read"
			}
			if g.Permission == "READ" && g.Grantee.ID == aCPolicy.Owner.ID {
				return "bucket-owner-read"
			}
		}
	case len(grants) == 3:
		for _, g := range grants {
			if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == "WRITE" {
				return "public-read-write"
			}
		}
	}
	return ""
}

func getAmzGrantACL(aCPolicy *accessControlPolicy) map[string][]string {
	grants := aCPolicy.AccessControlList.Grant
	res := map[string][]string{}

	for _, g := range grants {
		switch {
		case g.Permission == "READ":
			res["x-amz-grant-read"] = append(res["x-amz-grant-read"], "id="+g.Grantee.ID)
		case g.Permission == "WRITE":
			res["x-amz-grant-write"] = append(res["x-amz-grant-write"], "id="+g.Grantee.ID)
		case g.Permission == "READ_ACP":
			res["x-amz-grant-read-acp"] = append(res["x-amz-grant-read-acp"], "id="+g.Grantee.ID)
		case g.Permission == "WRITE_ACP":
			res["x-amz-grant-write-acp"] = append(res["x-amz-grant-write-acp"], "id="+g.Grantee.ID)
		case g.Permission == "FULL_CONTROL":
			res["x-amz-grant-full-control"] = append(res["x-amz-grant-full-control"], "id="+g.Grantee.ID)
		}
	}
	return res
}

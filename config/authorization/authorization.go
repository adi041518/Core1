package authorization

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/adi041518/Core1/config/db"
	"github.com/adi041518/Core1/config/jwt"
	"github.com/adi041518/Core1/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ctx context.Context = context.Background()

/*
here we are extracting the info from header
by trimming the prefix and if the header is valid only
it gets passed other and checks whether the bearer
Token is Invalid
*/
func ExtractTokenFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("invalid authorization format")
	}
	return tokenString, nil
}

/*
Verify user exists  checks the collection of user
sort according to the decreasing order by the key of
Updated At
*/
func VerifyUserExists(collectionName, code string) error {
	collection := db.OpenCollections(collectionName)
	filter := bson.M{"code": code}
	user := bson.M{}
	err := db.FindOne(ctx, collection, filter, user)
	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}
	// isActive, ok := user["isActive"].(bool)
	// if !ok {
	// 	return fmt.Errorf("invalid user data format (isActive missing or not boolean)")
	// }
	// if !isActive {
	// 	return fmt.Errorf("user is inactive")
	// }
	isActive, ok := user["isActive"].(bool)
	if !ok {
		isActive = true // assume user is active if field missing
	}
	if !isActive {
		return fmt.Errorf("user is inactive")
	}
	return nil
}

/*
here
1.first the extraction takes place
2.Validation of token takes place
3.Verify tenant is existing at the tenant level
4.if the collection is other than tenants then it gets verify user exists function will be
passed.
*/
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenString, err := ExtractTokenFromHeader(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		claims, err := jwt.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		log.Println(claims.Collection)
		if err := VerifyUserExists(claims.Collection, claims.Code); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.Set("code", claims.Code)
		c.Set("email", claims.Email)
		c.Set("roleCode", claims.RoleCode)
		c.Set("collection", claims.Collection)
		c.Set("tenantId", claims.TenantId)
		c.Set("isSuperAdmin", claims.IsSuperAdmin)
		c.Next()
	}
}

/*
Here the cors middleware takes place
*/
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // or specific domain
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

/*
* Extract roleCode from the request
 */
func GetRoleCode(c *gin.Context) (string, error) {
	roleCodeValue, exists := c.Get("roleCode")
	if !exists {
		log.Println("Unable to get the code key from claims")
		return "", errors.New(util.ROLE_CODE_KEY_NOT_FOUND)
	}

	roleCode, ok := roleCodeValue.(string)
	if !ok {
		log.Println("Type assertion error for roleCode")
		return "", errors.New(util.ROLE_CODE_VALUE_NOT_FOUND)
	}

	return roleCode, nil
}

/*
* Get role document for the given roleCode
* Find for the document and pass to extract privileges
 */
func GetRoleDocument(ctx context.Context, roleCode string) (map[string]interface{}, error) {
	coll := util.RoleCollection
	log.Println("coll:", coll)
	roleColl := db.OpenCollections(coll)

	filter := bson.M{
		"roleCode": roleCode,
	}
	log.Println("filter: ", filter)
	roleDoc := make(map[string]interface{})
	err := db.FindOne(ctx, roleColl, filter, &roleDoc)
	if err != nil {
		log.Println("Error fetching role document:", err)
		return nil, err
	}

	return roleDoc, nil
}

/* Extract privileges from RoleDocument from the document
* Check for the data and then pass to []map[string]interface{}
 */
func ExtractPrivileges(roleDoc map[string]interface{}) ([]map[string]interface{}, error) {

	privRaw, ok := roleDoc["privileges"]
	if !ok || privRaw == nil {
		return nil, errors.New(util.PRIVILEGES_DATA_REQUIRED)
	}

	fmt.Printf("DEBUG => privileges raw type = %T\n", privRaw)
	fmt.Printf("DEBUG => privileges raw value = %#v\n", privRaw)

	// CASE 1: primitive.A → Mongo Array
	if arr, ok := privRaw.(primitive.A); ok {
		privs := make([]map[string]interface{}, 0, len(arr))

		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				log.Println("Invalid privilege entry in primitive.A:", item)
				continue
			}
			privs = append(privs, m)
		}

		if len(privs) == 0 {
			return nil, errors.New(util.PRIVILEGES_DATA_REQUIRED)
		}
		return privs, nil
	}

	return nil, errors.New(util.PRIVILEGES_DATA_REQUIRED)
}

/*
* Go to the privileges and findModuleName
* Check whether the moduleName and dbModule are same
* If found same,go to the access []string then find for the access list and give access
 */
func HasAccessForPrivileges(privileges []map[string]interface{}, moduleName string, access string) (bool, string, []string, error) {
	for _, priv := range privileges {

		dbModule, _ := priv["module"].(string)

		if dbModule == moduleName {

			var accessList []string

			if alPrim, ok := priv["access"].(primitive.A); ok {
				for _, a := range alPrim {
					if s, ok := a.(string); ok {
						accessList = append(accessList, s)
					}
				}
				log.Println("access: ", alPrim)
				fmt.Printf("DEBUG => access raw type = %T\n", alPrim)
				fmt.Printf("DEBUG =>access raw value = %#v\n", alPrim)
			}

			if len(accessList) == 0 {
				return false, dbModule, nil,
					errors.New("each privilege must include at least one access right")
			}

			for _, aStr := range accessList {
				if aStr == access {
					return true, dbModule, accessList, nil
				}

			}

			return false, dbModule, accessList,
				fmt.Errorf("access '%s' not allowed for module '%s'", access, moduleName)
		}
	}

	return false, "", nil,
		fmt.Errorf("module '%s' not found in privileges", moduleName)
}

/*
* Extract roleCode from the context
* Get the document based on the roleCode
* Check for the privileges
* validate privileges and access whether they are valid or not will be checked here,Validate the access
 */
func Authorize(moduleName string, access string) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.GetString("code")
		log.Println("code from context: ", code)
		roleCode := c.GetString("roleCode")
		log.Println("roleCode from context: ", roleCode)
		ctxCollection := c.GetString("collection")
		log.Println("collection from context: ", ctxCollection)
		isSuperAdmin := c.GetBool("isSuperAdmin")
		log.Println("isSuperAdmin from context: ", isSuperAdmin)
		roleCode, err := GetRoleCode(c)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		roleDoc, err := GetRoleDocument(ctx, roleCode)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Extract privileges using the helper
		privileges, err := ExtractPrivileges(roleDoc)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		ok, foundModule, accessList, err := HasAccessForPrivileges(privileges, moduleName, access)
		if !ok {
			c.JSON(400, gin.H{
				"error":       err.Error(),
				"module":      foundModule,
				"access_list": accessList,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

package coreServices

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adi041518/Core1/util"

	"github.com/adi041518/Core1/config/db"
	"github.com/adi041518/Core1/config/redis"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var ctx context.Context = context.Background()

/*
Here It is Used for Generating EmpCode for
unique like tenants,patients like
based on the collection name we differ the prefix.
*/
func GenerateEmpCode(collName string) (string, error) {
	// Define prefix and number width for each collection
	var prefix string
	width := 4 // e.g. T0001 → 4 digits
	var sortField string = "code"
	switch collName {
	case "GUARDIAN":
		prefix = "G"
	case "BILL":
		prefix = "B"
	case "TEST_REPORT":
		prefix = "TR"
	case "PRESCRIPTION":
		prefix = "PRE"
	case "PHARMACIST":
		prefix = "PH"
	case "MEDICINES":
		prefix = "MED"
	case "DOCTOR_TIMESLOTS":
		prefix = "DT"
	case "APPOINTMENT":
		prefix = "A"
	case "MEDICAL_RECORD":
		prefix = "M"
	case "NURSE":
		prefix = "N"
	case "PATIENT":
		prefix = "P"
	case "RECEPTIONIST":
		prefix = "RE"
	case "DOCTOR":
		prefix = "D"
	case "HOSPITAL":
		prefix = "H"
	case "TENANT":
		prefix = "T"
	case "SUPERADMIN":
		prefix = "S"
	case "CONSENT":
		prefix = "C"
	case "ROLE":
		prefix = "R"
		sortField = "roleCode"
	case "TEST":
		prefix = "TE"
	default:
		return "", fmt.Errorf("unsupported collection: %s", collName)
	}

	collection := db.OpenCollections(collName)
	// Find last document sorted by code descending
	opts := options.FindOne().SetSort(bson.D{{Key: sortField, Value: -1}})
	var lastDoc bson.M

	err := collection.FindOne(ctx, bson.M{}, opts).Decode(&lastDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Start fresh
			return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
		}
		return "", err
	}
	// Extract last code

	codeVal, ok := lastDoc[sortField].(string)
	if !ok || codeVal == "" {
		return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
	}
	log.Println("code", codeVal)

	// Extract numeric part (e.g., T0005 → 5)
	re := regexp.MustCompile(`(\d+)$`)
	matches := re.FindStringSubmatch(codeVal)
	if len(matches) < 2 {
		return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
	}

	lastNum, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Sprintf("%s%0*d", prefix, width, 1), nil
	}

	newNum := lastNum + 1
	newCode := fmt.Sprintf("%s%0*d", prefix, width, newNum)
	return newCode, nil
}

func IsPhoneNumberExists(collName string, phone string) (bool, error) {
	collection := db.OpenCollections(collName)
	filter := bson.M{"phone": phone}
	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/*
* Check whether key exists in data
* Check for the type of data, and value stored at the field
* Trim and store in data
 */
func GetTrimmedString(data map[string]interface{}, key string) error {
	raw, exists := data[key]
	if !exists {
		return fmt.Errorf("%s missing field", key)
	}
	v, ok := raw.(string)
	if !ok {
		return fmt.Errorf("%s invalid type", key)
	}
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return fmt.Errorf("%s key has empty value", key)
	}
	data[key] = trimmed
	return nil
}

/*
Here It Verify Whether the Email is present in The Database.
*/
func IsEmailExists(collName string, email string) (bool, error) {
	collection := db.OpenCollections(collName)
	filter := bson.M{"email": email}
	emailcount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		log.Println("Error while countingDocument with email provided: ", err)
		return false, err
	}
	return emailcount > 0, err
}

/*
function for normalizing the Email
*/
func NormalizeEmail(email string) string {
	loweredEmail := strings.ToLower(email)
	return loweredEmail
}

/*
function for normalizing the phone Number
*/
func NormalizePhoneNumber(phone string) string {
	trimmedPhone := strings.TrimSpace(phone)
	if strings.Contains(trimmedPhone, " ") {
		return ""
	}
	return trimmedPhone
}

/*
Function For Phone Number Validation
*/
func IsPhoneNumberValid(phone string) bool {
	trimmedPhone := strings.TrimSpace(phone)
	rephone := strings.ReplaceAll(trimmedPhone, " ", "")
	re := regexp.MustCompile(`^(\+91)?[6-9]\d{9}$`)
	check := re.MatchString(rephone)
	return check
}

/*
Changes the DOB of any form into A Single DOB form
and make it parse and format into our style of DOB
checkes it its done return error if it is any invalid format
*/
func NormalizeDate(dobStr string) (string, error) {
	formats := []string{
		"2006-01-02",
		"02-01-2006",
		"02/01/2006",
		"2006/01/02",
	}

	dobStr = strings.TrimSpace(dobStr)

	var dob time.Time
	var err error

	for _, format := range formats {
		dob, err = time.Parse(format, dobStr)
		if err == nil {
			return dob.Format("2006-01-02"), nil
		}
	}
	return "", errors.New(util.INVALID_DATE_FORMAT)
}

// /*
//   - UserFetch
//     */
func UserFetch(ctx *gin.Context) (interface{}, error) {
	//interface
	code, exists := ctx.Get("code")
	if !exists {
		log.Println("Error while fetching from context")
		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
	}
	//convert to string
	codeStr, exist := code.(string)
	if !exist {
		log.Println("Error while converting from mongo collection to string")
	}

	claimsCollection, exists := ctx.Get("collection")
	if !exists {
		log.Println("Error while fetching from context")
		return nil, errors.New(util.ERROR_WHILE_FETCH_FROM_CONTEXT)
	}

	collectionStr, exist := claimsCollection.(string)
	if !exist {
		log.Println("Error while converting from mongo collection to string")
	}

	var user bson.M
	collection := db.OpenCollections(collectionStr)
	filter := bson.M{"user_id": codeStr}

	err := db.FindOne(ctx, collection, filter, user)
	if err != nil {
		log.Println("Error while finding a document")
		return nil, errors.New(util.ERR_NO_DOC_FOUND)
	}
	return user, nil
}

/*
* Fetch user by code
* Fetch from cache either exist return true nor false
* If true return user ,if not go to db
* Fetch from db that return error
* If no doc found nor fetching error return error ,if not bind with the varibale
* Set the value into the cache and then return the bind with the variable
 */
func FetchUserByCode(ctx *gin.Context, code string, collectionStr string) (interface{}, error) {

	collection := db.OpenCollections(collectionStr)

	user := make(map[string]interface{})
	filter := bson.M{"code": code}
	exists, err := redis.GetCache(ctx, code, &user)
	if err != nil {
		return nil, err
	}
	if !exists {
		err := db.FindOne(ctx, collection, filter, user)
		if err != nil {
			log.Println("Database fetch failed:", err)
			return nil, err

		}
		err = redis.SetCache(ctx, code, user)
		if err != nil {
			log.Println("Failed to set cache:", err)
			return nil, err
		}
	}
	return user, nil
}

/*
* Generate a random otp upto 999999
 */
func GenerateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

/*
 */
func SendOTPToMail(to, subject, body string) error {
	from := os.Getenv("SMTP_FROM")
	username := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	message := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n%s",
		from, to, subject, body,
	))

	auth := smtp.PlainAuth("", username, password, smtpHost)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, message)
}

func IscodeExists(collName string, code string) (bool, error) {
	collection := db.OpenCollections(collName)
	filter := bson.M{"code": code}
	emailcount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return emailcount > 0, err
}

/*
Checker validates email and phone number formats.
It also checks the database to ensure both fields do not already exist.
Returns an error if any validation rule fails.
*/
func Checker(Email string, Phone string, collName string) error {
	email := NormalizeEmail(Email)
	if email == "" {
		return errors.New(util.PROVIDED_EMPTY_EMAIL)
	}
	emailsCount, emailError := IsEmailExists(collName, Email)
	if emailError != nil {
		return emailError
	}
	if emailsCount == true {
		log.Println("Email Exists triggered")
		return errors.New(util.USER_EXISTING_EMAIL)
	}
	modifiedPhoneNumber := NormalizePhoneNumber(Phone)
	if modifiedPhoneNumber == "" {
		return errors.New(util.PROVIDED_EMPTY_PHONE_NUMBER)
	}
	Phone = modifiedPhoneNumber
	check := IsPhoneNumberValid(Phone)
	if check == false {
		return errors.New(util.PHONE_NUMBER_VALIDATION)
	}
	phoneNumbersCount, phoneNumberError := IsPhoneNumberExists(collName, Phone)
	if phoneNumberError != nil {
		return phoneNumberError
	}
	if phoneNumbersCount == true {
		log.Println("PhoneNo already exists")
		return errors.New(util.PHONE_NUMBER_ALREADY_EXISTS)
	}
	return nil
}

/*
* check for mandatory fields exists or not
* If not return error,else check for the type of field stored
* Trim and append to data
 */
func ValidateUserInput(data map[string]interface{}) error {
	fields := []string{"name", "email", "phoneNo", "dob", "roleCode"}
	for _, f := range fields {
		if err := GetTrimmedString(data, f); err != nil {
			log.Println("Error from getTrimmedString:", err)
			return err
		}
	}
	return nil
}

/*
FetchRoleById retrieves a role by its roleCode.
Steps:
1. Check Redis cache
2. If missing, fetch from MongoDB
3. Repopulate cache
*/
func FetchRoleById(c *gin.Context, roleCode string) (map[string]interface{}, error) {

	if strings.TrimSpace(roleCode) == "" {
		return nil, errors.New(util.ROLE_CODE_IS_EMPTY)
	}
	key := util.RoleKey + roleCode

	var cached map[string]interface{}
	found, err := redis.GetCache(c, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	collection := db.OpenCollections(util.RoleCollection)
	filter := bson.M{"roleCode": roleCode}
	role := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, role)
	if err != nil {
		return nil, errors.New(util.ERR_WHILE_FETCHING_ROLE)
	}

	return role, nil
}
func FetchCollectionFromRoleDoc(c *gin.Context, roleCode string) (string, error) {
	roleDoc, err := FetchRoleById(c, roleCode)
	if err != nil {
		log.Println("Error from FetchRolebyId", err)
		return "", err
	}
	collection, ok := roleDoc["roleName"].(string)
	if !ok {
		return "", errors.New("invalid roleName")
	}
	return collection, nil
}
func CheckerAndGenerateUserCodes(c *gin.Context, collection, email, phone string) (string, string, error) {

	if err := Checker(email, phone, collection); err != nil {

		log.Println("Error from checker function:", err)
		return "", "", err
	}

	code, err := GenerateEmpCode(collection)
	if err != nil {
		log.Println("Error from GenerateEmpCode:", err)
		return "", "", err
	}
	if collection == "SUPERADMIN" {
		return code, "SYSTEM", nil
	}
	userCodeVal, exists := c.Get("code")
	if !exists {
		log.Println("Error unable to get the code from the context")
		return "", "", errors.New("missing creator code")
	}

	createdBy := userCodeVal.(string)

	return code, createdBy, nil
}
func FetchTenantId(ctx *gin.Context, code string) (string, error) {
	collection := db.OpenCollections(util.HospitalCollection)
	filter := bson.M{"code": code}
	result := make(map[string]interface{})
	err := db.FindOne(ctx, collection, filter, result)
	if err != nil {
		return "", err
	}
	codeVal, ok := result["createdBy"]
	if !ok {
		return "", errors.New("tenantiD doesnt exist")
	}
	tenantid := codeVal.(string)
	log.Println(tenantid)
	return tenantid, nil

}
func CalculateAge(date_of_birth string) (int, error) {

	layouts := []string{"02-01-2006", "02/01/2006", "2006-01-02"}
	var birth_date time.Time
	var err error
	for _, layout := range layouts {
		birth_date, err = time.Parse(layout, date_of_birth)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, fmt.Errorf("invalid date format: please use DD-MM-YYYY or DD/MM/YYYY")
	}
	now := time.Now()
	age := now.Year() - birth_date.Year()
	if now.YearDay() < birth_date.YearDay() {
		age--
	}
	return age, nil
}
func GenerateAndHashOTP(data map[string]interface{}) (string, error) {

	otp := GenerateOTP()
	expiry := time.Now().Add(10 * time.Minute)
	data["otpExpiry"] = expiry
	log.Println(otp)
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Unable to bcrypt the otp")
		return "", errors.New(util.FAILED_TO_HASH_OTP)
	}
	log.Println(string(hashedOTP))
	data["password"] = string(hashedOTP)
	return otp, nil
}

/*
PrepareTenant formats and validates Tenant data.
Normalizes the DOB, sets default fields, and populates metadata like timestamps.
Used before inserting the record in the database.
*/
func PrepareUser(data map[string]interface{}, code string, createdBy string, tenantId string) error {

	dob, _ := data["dob"].(string)
	modifiedDob, err := NormalizeDate(dob)
	if err != nil {
		log.Println("Error from normalize function", err)
		return err
	}

	data["_id"] = primitive.NewObjectID()
	data["dob"] = modifiedDob
	data["code"] = code
	data["loginAttempts"] = 0
	data["reset"] = true
	data["isActive"] = false
	data["isBlocked"] = false
	data["tenantId"] = tenantId
	data["createdBy"] = createdBy
	data["updatedBy"] = createdBy
	data["createdAt"] = time.Now()
	data["updatedAt"] = time.Now()
	return nil
}
func SaveUserToDB(collection string, data map[string]interface{}) (primitive.ObjectID, error) {
	coll := db.OpenCollections(collection)
	res, err := db.CreateOne(context.Background(), coll, data)
	if err != nil {
		log.Println("Error from CreateOne function:", err)
		return primitive.NilObjectID, err
	}
	log.Println(res.InsertedID)
	return res.InsertedID.(primitive.ObjectID), nil
}

/*
* Insert into the loginRecord
 */
func CreateLoginRecord(ctx context.Context, role string, code string, email string, phone string, password string) error {

	loginCollection := db.OpenCollections("LOGIN")
	filter := bson.M{
		"$or": []bson.M{
			{"email": email},
			{"phoneNo": phone},
		},
	}
	log.Println(filter)
	existing := make(map[string]interface{})
	err := db.FindOne(ctx, loginCollection, filter, &existing)
	log.Println("existing: ", existing)
	log.Println(err)
	if err == nil && len(existing) > 0 {
		if v, ok := existing["email"].(string); ok && v == email {
			return errors.New(util.USER_EXISTING_EMAIL_IN_LOGIN)
		}

		if v, ok := existing["phoneNo"].(string); ok && v == phone {
			return errors.New(util.USER_EXISTING_PHONE_NUMBER_IN_LOGIN)
		}
		log.Println("Already exists in db: ", err)
		return errors.New("Already exists in loginCollection")
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		log.Println("Error from FindOne (unexpected):", err)
		return fmt.Errorf("error checking existing login: %w", err)
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		login := bson.M{
			"code":       code,
			"collection": role,
			"email":      email,
			"phoneNo":    phone,
			"password":   password,
		}

		_, err = db.CreateOne(ctx, loginCollection, login)
		if err != nil {
			log.Println("Error from createOne: ", err)
			return fmt.Errorf("failed to createOne login record: %v", err)
		}
		return nil
	}
	return nil
}

func CacheUserInRedis(c *gin.Context, code string, key string, data map[string]interface{}, collection string) error {

	err := redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from SetCache:", err)
		return errors.New("Error from setCache")
	}
	return nil
}
func GetTenantIdFromContext(c *gin.Context) (string, error) {
	val := ""
	tenantIdVal, ok := c.Get("tenantId")
	if !ok {
		log.Println("Error while fetching tenantId from token")
		return val, errors.New(util.UNABLE_TO_FETCH_TENANT_ID_FROM_CONTEXT)
	}
	tenantId, ok := tenantIdVal.(string)
	if !ok {
		log.Println("Error while type converting from interface to string tenantId ")
		return val, errors.New(util.ERR_WHILE_FETCHING_TENANT_ID_OF_TYPE_STRING)
	}
	return tenantId, nil
}
func IsSuperAdmin(c *gin.Context) (bool, error) {
	IsSuperAdmin, ok := c.Get("isSuperAdmin")
	if !ok {
		log.Println("error from issuperAdmin")
		return false, errors.New(util.UNABLE_TO_FETCH_IS_SUPER_ADMIN_FROM_CONTEXT)
	}
	issuperadmin := IsSuperAdmin.(bool)
	return issuperadmin, nil
}

func GetFromContext[T any](c *gin.Context, key string) (T, error) {
	val, ok := c.Get(key)
	if !ok {
		var zero T
		return zero, fmt.Errorf("key '%s' not found in context", key)
	}

	value, ok := val.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("type assertion failed for key '%s'", key)
	}

	return value, nil
}

func FetchByCodeFromCache(c *gin.Context, key string, isSuperAdmin bool, tenantId string, code string, ctxCollection string) (map[string]interface{}, bool, error) {

	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	if err != nil || !exists {
		return nil, false, err
	}

	if err := HasAccess(isSuperAdmin, ctxCollection, tenantId, code, cached); err != nil {
		return nil, false, err
	}

	return cached, true, nil
}

func HasAccess(isSuperAdmin bool, cxtCollection string, tenantId string, code string, doc map[string]interface{}) error {

	if isSuperAdmin {
		return nil
	}

	tenantIdFromDoc, ok := doc["tenantId"].(string)
	if !ok {
		return errors.New(util.MISSING_TENANT_ID_FROM_DOCUMENT)
	}

	if cxtCollection == util.TenantCollection {
		if tenantId != tenantIdFromDoc {
			return errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
		}
	}

	createdByFromDoc, ok := doc["createdBy"].(string)
	if !ok {
		return errors.New(util.MISSING_CREATED_BY_IN_DOCUMENT)
	}

	if cxtCollection == util.HospitalCollection {
		if code != createdByFromDoc {
			return errors.New(util.INVALID_USER_TO_ACCESS)
		}
	}

	return nil
}

func CanAccess(userData, record map[string]interface{}, tenantId string, code string, collFromContext string, isSuperAdmin bool) error {
	log.Println("record: ", record)

	if isSuperAdmin {
		return nil
	}

	if collFromContext == util.TenantCollection {
		if record["tenantId"].(string) != tenantId {
			return errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
		}
		return nil
	}

	if collFromContext == util.HospitalCollection {
		if record["hospitalId"].(string) != code {
			return errors.New(util.HOSPITAL_ADMIN_DOESNOT_HAVE_ACCESS)
		}
		return nil
	}

	log.Println("userData: ", userData["createdBy"].(string))
	log.Println("hospitalId: ", record["hospitalId"].(string))
	if userData["createdBy"].(string) != record["hospitalId"].(string) {
		return errors.New(util.INVALID_USER_TO_ACCESS)
	}

	return nil
}

func CheckCacheAccess(c *gin.Context, key string, collFromContext string, userData map[string]interface{}, tenantId, code string, isSuperAdmin bool) (map[string]interface{}, bool, error) {

	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	if err != nil || !exists {
		return nil, false, nil
	}

	if err := CanAccess(userData, cached, tenantId, code, collFromContext, isSuperAdmin); err != nil {
		return nil, true, err
	}

	return cached, true, nil
}

func CheckForEmailAndPhoneNo(c *gin.Context, collection *mongo.Collection, data map[string]interface{}) error {
	fields := []string{"email", "phoneNo"}
	for _, fieldName := range fields {
		fieldVal, ok := data[fieldName]
		if ok {
			fieldStr, ok := fieldVal.(string)
			filter := bson.M{
				fieldName: fieldStr,
			}
			result := make(map[string]interface{})
			if ok {
				err := db.FindOne(c, collection, filter, &result)
				if err == nil {
					if fieldName == "email" {
						return errors.New(util.USER_EXISTING_EMAIL)
					} else {
						return errors.New(util.PHONE_NUMBER_ALREADY_EXISTS)
					}
				}
				if err != mongo.ErrNoDocuments {
					return err
				}
			}
		}

	}
	return nil
}

/*
* Trim fields if they exists and fix them into the input data
 */
func TrimIfExists(data map[string]interface{}, key string) error {
	if _, exists := data[key]; exists {
		err := GetTrimmedString(data, key)
		if err != nil {
			log.Printf("Error trimming %s: %v", key, err)
			return err
		}
	}
	return nil
}

/*
* If DOB field exists then trim and normalize it
* Insert into the input field
 */
func HandleDOB(data map[string]interface{}) error {
	raw, exists := data["dob"]
	if !exists {
		return nil
	}

	dobStr, ok := raw.(string)
	if !ok {
		return errors.New(util.INVALID_DOB_FORMAT)
	}

	if err := GetTrimmedString(data, "dob"); err != nil {
		return err
	}

	normalized, err := NormalizeDate(dobStr)
	if err != nil {
		return err
	}

	data["dob"] = normalized
	return nil
}

/*
* Include all fields provided and extra field to modify into the input data provided
* Make it as update filter
 */
func BuildUpdateFilter(data map[string]interface{}, code string) map[string]interface{} {
	// data["createdBy"] = createdBy
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	updateFilter := bson.M{"$set": data}
	return updateFilter
}

func AttachNamesFromRedis(c context.Context, data map[string]interface{}) (map[string]map[string]interface{}, []string) {

	fields := []string{
		"patientId",
		"doctorId",
		"hospitalId",
		"tenantId",
		"createdBy",
		"updatedBy",
	}

	cachedResults := make(map[string]map[string]interface{})
	var keys []string
	for _, field := range fields {

		key, ok := data[field].(string)
		if !ok || key == "" {
			continue
		}
		keys = append(keys, key)
		var cached map[string]interface{}

		found, err := redis.GetCache(c, key, &cached)
		if err != nil || !found {
			continue
		}

		cachedResults[key] = cached

		if name, ok := cached["name"]; ok {
			nameField := field[:len(field)-2] + "Name"
			data[nameField] = name
		}
	}

	return cachedResults, keys
}

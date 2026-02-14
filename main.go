package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	jwtware "github.com/gofiber/jwt/v3"
	_ "github.com/lib/pq"
)

var jwtSecret = []byte("your_secret_key")

const (
	host     = "localhost"          // or the Docker service name if running in another container
	port     = 5432                 // default PostgreSQL port
	user     = "wearlab"            // as defined in docker-compose.yml
	password = "wearlabbro30102001" // as defined in docker-compose.yml
	dbname   = "wearlabdatabase"    // as defined in docker-compose.yml
)

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Product struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Defect      string   `json:"defect"`
	Type        string   `json:"type"`
	Waist       int      `json:"waist"`
	Length      int      `json:"length"`
	Chest       int      `json:"chest"`
	Owner       int      `json:"owner"`
	Status      string   `json:"status"`
	Price       int      `json:"price"`
	SalePrice   int      `json:"saleprice"`
	Image       []string `json:"image"`
	Create_Date string   `json:"createdate"`
	Update_Date string   `json:"updatedate"`
	Owner_Name  string   `json:"ownername"`
}

type ProductListResponse struct {
	Total    int       `json:"total"`
	Products []Product `json:"products"`
}

type Owner struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Type struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Supplier struct {
	ID   int
	Name string
}

type DayOff struct {
	ID   int    `json:"id"`
	Date string `json:"date"`
	Uid  int    `json:"uid"`
}

type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Phone     int    `json:"phone"`
	Address   string `json:"address"`
	Role      string `json:"role"`
	Country   string `json:"country"`
	Zipcode   int    `json:"zipcode"`
}

type Item struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Token struct {
	Token string `json:"token"`
}

var db *sql.DB

func main() {
	// Connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open a connection
	sdb, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	db = sdb

	// Check the connection to make sure
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Use(cors.New())

	app.Post("/login", loginHandler)

	app.Get("/product/filter", getProductWithFilterHandler)
	app.Get("/product/:id", getProductByIdHandle)
	app.Get("/product", getProductsHandler)
	app.Get("/owner", getOwnersHandler)
	app.Put("/owner/:id", updateOwnerHandler)
	app.Post("/owner", createOwnerHandler)
	app.Get("/type", getTypesHandler)
	app.Get("/status", getStatusHandler)
	app.Post("/user", createUserHandler)
	app.Get("/users", getUsersHandler)

	// Protected routes for /product only
	productGroup := app.Group("/product", jwtware.New(jwtware.Config{
		SigningKey: jwtSecret,
	}))

	productGroup.Post("/", createProductHandler)
	productGroup.Put("/:id", updateProductHandle)
	productGroup.Delete("/:id", deleteProductHandler)

	// If you also want to protect these, move them to another group:
	ownerGroup := app.Group("/owner", jwtware.New(jwtware.Config{
		SigningKey: jwtSecret,
	}))

	ownerGroup.Put("/:id", updateOwnerHandler)
	ownerGroup.Post("/", createOwnerHandler)

	// Start Fiber and Socket.IO
	app.Listen(":8080")

	// app.Listen(":8080")

	// fmt.Println("Successfully connected!")

	// err = createProduct(&Product{Name: "Go product", Price: 220})

	// product, err := getProductById(2)

	// for choice 1
	// err = updateProduct(1, &Product{Name: "New name", Price: 310})

	//for choice 2
	// product, err := updateProduct(5, &Product{Name: "New name id 3", Price: 320})

	// err = deleteProduct(1)

	// product, err := getProducts()

	// err = addProductAndSupplier("CPF", "Potato Chips", 25)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("Create Successful !")

	// fmt.Println("Create Successful !", product)
}

func loginHandler(c *fiber.Ctx) error {
	req := new(Login)

	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	token, err := login(req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
	})
}

func getUsersHandler(c *fiber.Ctx) error {
	users, err := getUsers()

	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.JSON(users)
}

func getProductByIdHandle(c *fiber.Ctx) error {
	// Convert the "id" parameter from the request to an integer
	id, err := strconv.Atoi(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid product ID")
	}

	// Call the getProductById function to retrieve the product
	product, err := getProductById(id)

	if err != nil {
		// Check if the error is due to no rows being found
		if err.Error() == fmt.Sprintf("no product found with id %d", id) {
			return c.Status(fiber.StatusNotFound).SendString(err.Error())
		}
		// Handle other possible errors
		return c.Status(fiber.StatusInternalServerError).SendString("An error occurred while retrieving the product")
	}

	// If the product is found, return it as a JSON response
	return c.JSON(product)
}

func createProductHandler(c *fiber.Ctx) error {
	product := new(Product)

	if err := c.BodyParser(product); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	err := createProduct(product)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.SendString("Create Product Successfully.")
}

func createOwnerHandler(c *fiber.Ctx) error {
	owner := new(Owner)

	if err := c.BodyParser(owner); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	err := createOwner(owner)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.SendString("Create New Owner Successfully.")
}

func createUserHandler(c *fiber.Ctx) error {
	user := new(User)

	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	err := createUser(user)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.SendString("Create User Successfully.")
}

func deleteProductHandler(c *fiber.Ctx) error {
	// Parse the uid parameter
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Product ID")
	}

	// Attempt to delete the day off
	err = deleteProduct(id)
	if err != nil {
		if err.Error() == "no record found to delete" {
			return c.Status(fiber.StatusNotFound).SendString("No matching record found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete product")
	}

	return c.SendString("Product deleted successfully.")
}

func updateProductHandle(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	product := new(Product)

	if err := c.BodyParser(product); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	updateProduct, err := updateProduct(id, product)

	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.JSON(updateProduct)
}

func updateOwnerHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ID")
	}

	var owner Owner
	if err := c.BodyParser(&owner); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	o, err := updateOwner(id, &owner)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.JSON(o)
}

func updateMultipleProductsHandle(c *fiber.Ctx) error {
	var products []Product

	if err := c.BodyParser(&products); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	var updatedProducts []Product

	for _, product := range products {
		if product.ID == 0 {
			return c.Status(fiber.StatusBadRequest).SendString("Product ID is required for update")
		}

		updated, err := updateProduct(product.ID, &product)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to update product ID %d: %v", product.ID, err))
		}

		updatedProducts = append(updatedProducts, updated)
	}

	return c.JSON(updatedProducts)
}

func getProductWithFilterHandler(c *fiber.Ctx) error {
	// Get string filters directly (no need to convert)
	status := c.Query("status")
	prodType := c.Query("type")
	name := c.Query("name")

	// Fetch "limit" and "offset" from query parameters
	limit, err := strconv.Atoi(c.Query("limit", "15")) // Default to 15 if not provided
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Limit")
	}

	offset, err := strconv.Atoi(c.Query("offset", "0")) // Default to 0 if not provided
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Offset")
	}

	// Fetch products with the parsed limit and offset
	products, total, err := getProductWithFilter(limit, offset, status, prodType, name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get products")
	}

	// Return paginated response with total count
	return c.JSON(fiber.Map{
		"products": products,
		"total":    total,
	})
}

func getProductsHandler(c *fiber.Ctx) error {
	// Fetch "limit" and "offset" from query parameters
	limit, err := strconv.Atoi(c.Query("limit", "15")) // Default to 15 if not provided
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Limit")
	}

	offset, err := strconv.Atoi(c.Query("offset", "0")) // Default to 0 if not provided
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Offset")
	}

	// Fetch products with the parsed limit and offset
	products, total, err := getProducts(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get products")
	}

	// Return paginated response with total count
	return c.JSON(fiber.Map{
		"products": products,
		"total":    total,
	})
}

func getOwnersHandler(c *fiber.Ctx) error {
	owners, err := getOwners()

	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.JSON(owners)
}

func getTypesHandler(c *fiber.Ctx) error {
	types, err := getTypes()

	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.JSON(types)
}

func getStatusHandler(c *fiber.Ctx) error {
	types, err := getStatus()

	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.JSON(types)
}

// func getDayOffsHandler(c *fiber.Ctx) error {
// 	uid, err := strconv.Atoi(c.Params("uid"))

// 	if err != nil {
// 		return c.SendStatus(fiber.StatusBadRequest)
// 	}

// 	dayOffs, err := getDayOffs(uid)

// 	if err != nil {
// 		return c.SendStatus(fiber.StatusBadRequest)
// 	}

// 	return c.JSON(dayOffs)
// }

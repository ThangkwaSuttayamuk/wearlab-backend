package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lib/pq"

	_ "github.com/lib/pq"
)

func login(login *Login) (string, error) {
	var dbUser User

	err := db.QueryRow(
		`SELECT id, email, password FROM public.user WHERE email=$1 AND password=$2`,
		login.Email, login.Password,
	).Scan(&dbUser.ID, &dbUser.Email, &dbUser.Password)

	if err != nil {
		return "", err
	}

	// Create JWT token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["email"] = dbUser.Email
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return t, nil
}

func getUsers() ([]User, error) {
	rows, err := db.Query("SELECT id, firstname, lastname FROM users")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []User

	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Firstname, &u.Lastname)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func createProduct(product *Product) error {
	currentTime := (time.Now())

	_, err := db.Exec(
		"INSERT INTO public.product(name, description, defect, type, waist, length, chest, owner, status, price, saleprice, image, createdate, updatedate) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14);",
		product.Name, product.Description, product.Defect, product.Type, product.Waist, product.Length, product.Chest, product.Owner, product.Status, product.Price, product.SalePrice, pq.Array(product.Image), currentTime, currentTime,
	)

	return err
}

func createOwner(owner *Owner) error {

	_, err := db.Exec(
		"INSERT INTO public.owner(name) VALUES ($1);",
		owner.Name,
	)

	return err
}

func createUser(user *User) error {
	_, err := db.Exec(
		"INSERT INTO public.users(firstname, lastname) VALUES ($1, $2);",
		user.Firstname, user.Lastname,
	)

	return err
}

func deleteProduct(id int) error {
	result, err := db.Exec(
		"DELETE FROM public.product WHERE id = $1 ", id,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no record found to delete")
	}

	return nil
}

func getProductById(id int) (Product, error) {
	var p Product

	row := db.QueryRow(`
		SELECT 
			p.id, p.name, p.description, p.defect, p.type, p.waist, p.length, p.chest, p.owner,
			p.status, p.price, p.saleprice, p.image, p.createdate, p.updatedate,
			o.name as ownername
		FROM 
			product p
		JOIN 
			owner o ON p.owner = o.id
		WHERE 
			p.id = $1;
	`, id)

	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Defect, &p.Type, &p.Waist, &p.Length, &p.Chest, &p.Owner,
		&p.Status, &p.Price, &p.SalePrice, pq.Array(&p.Image), &p.Create_Date, &p.Update_Date, &p.Owner_Name)

	if err != nil {
		if err == sql.ErrNoRows {
			return Product{}, fmt.Errorf("no product found with id %d", id)
		}
		return Product{}, err
	}

	return p, err
}

func getOwnerById(id int) (Owner, error) {
	var o Owner

	row := db.QueryRow(
		"SELECT id, name FROM public.owner WHERE id = $1;",
		id,
	)

	err := row.Scan(&o.ID, &o.Name)

	// Handle the case where no rows were found
	if err != nil {
		if err == sql.ErrNoRows {
			// Return a custom error indicating no product found
			return Owner{}, fmt.Errorf("no owner found with id %d", id)
		}
		// Return any other errors encountered during scanning
		return Owner{}, err
	}

	return o, err
}

func updateProduct(id int, product *Product) (Product, error) {
	var p Product
	currentTime := time.Now()

	row := db.QueryRow(
		`UPDATE public.product
		SET name = $1, description = $2, defect = $3, type = $4,
		    waist = $5, length = $6, chest = $7, owner = $8,
		    status = $9, price = $10, saleprice = $11,
		    image = $12, updatedate = $13
		WHERE id = $14
		RETURNING id, name, description, defect, type, waist, length, chest,
		          owner, status, price, saleprice, image, createdate, updatedate;`,
		product.Name, product.Description, product.Defect, product.Type,
		product.Waist, product.Length, product.Chest, product.Owner,
		product.Status, product.Price, product.SalePrice,
		pq.Array(product.Image),
		currentTime, id,
	)

	err := row.Scan(
		&p.ID, &p.Name, &p.Description, &p.Defect, &p.Type, &p.Waist, &p.Length,
		&p.Chest, &p.Owner, &p.Status, &p.Price, &p.SalePrice,
		pq.Array(&p.Image),
		&p.Create_Date, &p.Update_Date,
	)

	if err != nil {
		return Product{}, err
	}

	return p, nil
}

func updateOwner(id int, owner *Owner) (Owner, error) {
	var o Owner

	// Update the owner table (change name)
	row := db.QueryRow(
		"UPDATE public.owner SET name = $1 WHERE id = $2 RETURNING id, name;",
		owner.Name, id,
	)

	err := row.Scan(&o.ID, &o.Name)

	if err != nil {
		return Owner{}, err
	}

	return o, nil
}

func getProductWithFilter(limit, offset int, status, prodType, name string) ([]Product, int, error) {
	var (
		products     []Product
		args         []interface{}
		whereClauses []string
	)

	argID := 1

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.status = $%d", argID))
		args = append(args, status)
		argID++
	}
	if prodType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.type = $%d", argID))
		args = append(args, prodType)
		argID++
	}
	if name != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.name ILIKE $%d", argID))
		args = append(args, "%"+name+"%")
		argID++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM product p " + whereSQL
	var count int
	err := db.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	limitArg := argID
	offsetArg := argID + 1

	query := fmt.Sprintf(`
		SELECT 
			p.id, p.name, p.description, p.defect, p.type, p.waist, p.length, p.chest, p.owner,
			p.status, p.price, p.saleprice, p.image, p.createdate, p.updatedate,
			o.name as ownername
		FROM 
			product p
		JOIN 
			owner o ON p.owner = o.id
		%s
		ORDER BY p.id
		LIMIT $%d OFFSET $%d
	`, whereSQL, limitArg, offsetArg)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Defect, &p.Type, &p.Waist, &p.Length, &p.Chest, &p.Owner,
			&p.Status, &p.Price, &p.SalePrice, pq.Array(&p.Image), &p.Create_Date, &p.Update_Date, &p.Owner_Name)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return products, count, nil
}

func getProducts(limit int, offset int) ([]Product, int, error) {
	// Get total count
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM product").Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated products
	rows, err := db.Query(`
		SELECT 
			p.id, p.name, p.description, p.defect, p.type, p.waist, p.length, p.chest, p.owner,
			p.status, p.price, p.saleprice, p.image, p.createdate, p.updatedate,
			o.name as ownername
		FROM 
			product p
		JOIN 
			owner o ON p.owner = o.id
		ORDER BY p.id
		LIMIT $1 OFFSET $2;
	`, limit, offset)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Defect, &p.Type, &p.Waist, &p.Length, &p.Chest, &p.Owner,
			&p.Status, &p.Price, &p.SalePrice, pq.Array(&p.Image), &p.Create_Date, &p.Update_Date, &p.Owner_Name)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return products, count, nil
}

func countProducts() (int, error) {
	var count int

	err := db.QueryRow("SELECT COUNT(*) FROM product;").Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func getOwners() ([]Owner, error) {
	rows, err := db.Query("SELECT id, name FROM owner")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var owners []Owner

	for rows.Next() {
		var o Owner
		err := rows.Scan(&o.ID, &o.Name)
		if err != nil {
			return nil, err
		}
		owners = append(owners, o)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return owners, nil
}

func getTypes() ([]Type, error) {
	rows, err := db.Query("SELECT id, name FROM type")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var types []Type

	for rows.Next() {
		var t Type
		err := rows.Scan(&t.ID, &t.Name)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return types, nil
}

func getStatus() ([]Status, error) {
	rows, err := db.Query("SELECT id, name FROM status")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var allStatus []Status

	for rows.Next() {
		var s Status
		err := rows.Scan(&s.ID, &s.Name)
		if err != nil {
			return nil, err
		}
		allStatus = append(allStatus, s)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return allStatus, nil
}

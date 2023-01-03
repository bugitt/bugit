# Table "access"

```
  FIELD  | COLUMN  |   POSTGRESQL    |         MYSQL          
---------+---------+-----------------+------------------------
  ID     | id      | BIGSERIAL       | BIGINT AUTO_INCREMENT  
  UserID | user_id | BIGINT NOT NULL | BIGINT NOT NULL        
  RepoID | repo_id | BIGINT NOT NULL | BIGINT NOT NULL        
  Mode   | mode    | BIGINT NOT NULL | BIGINT NOT NULL        

Primary keys: id
```

# Table "access_token"

```
     FIELD    |    COLUMN    |     POSTGRESQL     |         MYSQL          
--------------+--------------+--------------------+------------------------
  ID          | id           | BIGSERIAL          | BIGINT AUTO_INCREMENT  
  UserID      | uid          | BIGINT             | BIGINT                 
  Name        | name         | TEXT               | LONGTEXT               
  Sha1        | sha1         | VARCHAR(40) UNIQUE | VARCHAR(40) UNIQUE     
  CreatedUnix | created_unix | BIGINT             | BIGINT                 
  UpdatedUnix | updated_unix | BIGINT             | BIGINT                 

Primary keys: id
```

# Table "lfs_object"

```
    FIELD   |   COLUMN   |      POSTGRESQL      |        MYSQL          
------------+------------+----------------------+-----------------------
  RepoID    | repo_id    | BIGINT               | BIGINT                
  OID       | oid        | TEXT                 | VARCHAR(191)          
  Size      | size       | BIGINT NOT NULL      | BIGINT NOT NULL       
  Storage   | storage    | TEXT NOT NULL        | LONGTEXT NOT NULL     
  CreatedAt | created_at | TIMESTAMPTZ NOT NULL | DATETIME(3) NOT NULL  

Primary keys: repo_id, oid
```

# Table "login_source"

```
     FIELD    |    COLUMN    |    POSTGRESQL    |         MYSQL          
--------------+--------------+------------------+------------------------
  ID          | id           | BIGSERIAL        | BIGINT AUTO_INCREMENT  
  Type        | type         | BIGINT           | BIGINT                 
  Name        | name         | TEXT UNIQUE      | VARCHAR(191) UNIQUE    
  IsActived   | is_actived   | BOOLEAN NOT NULL | BOOLEAN NOT NULL       
  IsDefault   | is_default   | BOOLEAN          | BOOLEAN                
  Config      | cfg          | TEXT             | TEXT                   
  CreatedUnix | created_unix | BIGINT           | BIGINT                 
  UpdatedUnix | updated_unix | BIGINT           | BIGINT                 

Primary keys: id
```


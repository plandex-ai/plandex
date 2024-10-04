### Subtask 1:  Update the addUser method to prevent adding users with duplicate usernames.

```js
class UserManager {
    constructor(users) {
        this.users = users;
    }

    addUser(user) {
        if (this.findUser(user.username)) {
            console.error(`User with username ${user.username} already exists`);
            return;
        }
        this.users.push(user);
    }

     findUser(username) {
         return this.users.find(u => u.username === username);
     }
}
```

### Subtask 2:  Modify the findUser method to log an error if the user is not found.

```js
class UserManager {
    constructor(users) {
        this.users = users;
    }

    addUser(user) {
        if (this.findUser(user.username)) {
            console.error(`User with username ${user.username} already exists`);
            return;
        }
        this.users.push(user);
    }

     findUser(username) {
         const user = this.users.find(u => u.username === username);
         if (!user) {
             console.error(`User with username ${username} not found`);
         }
     }
}
```

### Subtask 3:  Add a method to remove a user by username.

```js
class UserManager {
    constructor(users) {
        this.users = users;
    }

    addUser(user) {
        if (this.findUser(user.username)) {
            console.error(`User with username ${user.username} already exists`);
            return;
        }
        this.users.push(user);
    }

    findUser(username) {
        const user = this.users.find(u => u.username === username);
        if (!user) {
            console.error(`User with username ${username} not found`);
        }
    }

    removeUser(username) {
        const index = this.users.findIndex(u => u.username === username);
        if (index !== -1) {
            this.users.splice(index, 1);
        } else {
            console.error(`User with username ${username} not found`);
        }
    }
}
```

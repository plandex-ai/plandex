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
class UserManager {
    constructor(users) {
        this.users = users;
    }
    addUser(user) {
        this.users.push(user);
    }
    findUser(username) {
        return this.users.find(u => u.username === username);
    }
}
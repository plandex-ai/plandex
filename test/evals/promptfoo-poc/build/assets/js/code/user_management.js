pdx-1: class UserManager {
pdx-2:     constructor(users) {
pdx-3:         this.users = users;
pdx-4:     }
pdx-5: 
pdx-6:     addUser(user) {
pdx-7:         this.users.push(user);
pdx-8:     }
pdx-9: 
pdx-10:     findUser(username) {
pdx-11:         return this.users.find(u => u.username === username);
pdx-12:     }
pdx-13: }

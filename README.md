# Cribb

Cribb is a roommate management application designed to streamline apartment group coordination for chores and shared pantry management. The application is built with Angular 19 and Go 1.23, providing a comprehensive solution for household task management and inventory tracking.

## Features

### Authentication & User Management
- User registration and login system
- JWT-based authentication for secure access
- Create apartment groups
- Join existing apartment groups using group codes
- Secure logout functionality

### Chore Management
- Create individual or recurring chores for your apartment group
- Earn points for completing chores
- Set due dates and reminders
- Automated chore rotation for recurring tasks (e.g., weekly kitchen cleaning)
- Delete chores as needed

### Pantry Management
- Track shared pantry items and quantities
- Real-time updates for consumed items
- Automated notifications for depleted stock
- Monitor expiration dates

### Shopping List
- Add items from pantry or create new entries
- Group shopping lists for shared purchases
- Transfer items from shopping cart directly to pantry
- Track out-of-stock and expired items

## Technology Stack

### Frontend
- Angular CLI: 19.1.6
- Node: 20.16.0
- Package Manager: npm 10.8.1
- Angular: 19.1.5 (animations, common, compiler, forms, router)

### Backend
- Go 1.23.3
- JWT authentication
- No additional frameworks used

## Getting Started

### Prerequisites
- Node.js v20.16.0 or higher
- Go 1.23.3 or higher
- npm 10.8.1 or higher

### Installation

1. Clone the repository
```bash
git clone https://github.com/yourusername/cribb.git
cd cribb
```

2. Backend Setup
```bash
cd backend
go mod init cribb # Initialize module if not already done
go mod tidy # Install all dependencies
go run main.go
```

3. Frontend Setup
```bash
cd frontend
npm install
ng serve
```

The application will be available at `http://localhost:4200`

## Usage

1. **Register/Login**: Create an account or login with existing credentials
2. **Create/Join a Group**: 
   - Create a new apartment group and get a unique group code
   - Join an existing group using the group code
3. **Add Chores**: Create one-time or recurring chores for your group
4. **Manage Pantry**: Add and update items in your shared pantry
5. **Shopping List**: Create shopping lists and transfer items to pantry
6. **Logout**: Securely logout from your account

## Contributing

Contributions are welcome. Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## System Requirements

- Windows 10/11, macOS, or Linux
- 4GB RAM minimum
- 500MB disk space

## Support

For support or feedback, please open an issue in the GitHub repository.

## Authors

Developed for students and apartment dwellers to simplify shared living management.

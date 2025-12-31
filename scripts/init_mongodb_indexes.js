// MongoDB Index Creation Script for Questionarie Service
// Run this script using: mongosh "mongodb+srv://vanessa_db_user:2g0UPNE9e6lZ95x9@cluster-moova.s4lzypb.mongodb.net/?appName=cluster-moova" < init_mongodb_indexes.js

// Switch to the wemoova_questionnaires database
use wemoova_questionnaires;

print("Creating indexes for wemoova_questionnaires database...");

// ===== Collection: companies =====
print("Creating indexes for 'companies' collection...");
db.companies.createIndex({ "name": 1 });
db.companies.createIndex({ "created_at": -1 });

// ===== Collection: questionnaires =====
print("Creating indexes for 'questionnaires' collection...");
db.questionnaires.createIndex({ "created_by": 1 });
db.questionnaires.createIndex({ "is_active": 1 });
db.questionnaires.createIndex({ "created_at": -1 });
db.questionnaires.createIndex({ "title": 1 });

// ===== Collection: company_questionnaires =====
print("Creating indexes for 'company_questionnaires' collection...");
db.company_questionnaires.createIndex({ "company_id": 1 });
db.company_questionnaires.createIndex({ "questionnaire_id": 1 });
db.company_questionnaires.createIndex({ "period_start": 1, "period_end": 1 });
db.company_questionnaires.createIndex({ "is_active": 1 });
db.company_questionnaires.createIndex({ "company_id": 1, "is_active": 1 });
db.company_questionnaires.createIndex({ "assigned_by": 1 });
db.company_questionnaires.createIndex({ "assigned_at": -1 });

// ===== Collection: user_questionnaire_assignments =====
print("Creating indexes for 'user_questionnaire_assignments' collection...");
db.user_questionnaire_assignments.createIndex({ "user_id": 1 });
db.user_questionnaire_assignments.createIndex({ "company_questionnaire_id": 1 });
db.user_questionnaire_assignments.createIndex({ "status": 1 });
db.user_questionnaire_assignments.createIndex({ "user_id": 1, "status": 1 });
db.user_questionnaire_assignments.createIndex({ "company_questionnaire_id": 1, "status": 1 });
db.user_questionnaire_assignments.createIndex({ "assigned_by": 1 });
db.user_questionnaire_assignments.createIndex({ "assigned_at": -1 });
db.user_questionnaire_assignments.createIndex({ "completed_at": -1 });

// Compound index for preventing duplicate assignments
db.user_questionnaire_assignments.createIndex(
  { "user_id": 1, "company_questionnaire_id": 1 },
  { unique: true }
);

// ===== Collection: users_metadata =====
print("Creating indexes for 'users_metadata' collection...");
db.users_metadata.createIndex({ "company_id": 1 });
db.users_metadata.createIndex({ "supervisor_id": 1 });
db.users_metadata.createIndex({ "company_id": 1, "department": 1 });
db.users_metadata.createIndex({ "department": 1 });
db.users_metadata.createIndex({ "created_at": -1 });

print("All indexes created successfully!");

// Display created indexes
print("\n===== Created Indexes Summary =====");
print("\nCompanies indexes:");
printjson(db.companies.getIndexes());

print("\nQuestionnaires indexes:");
printjson(db.questionnaires.getIndexes());

print("\nCompany Questionnaires indexes:");
printjson(db.company_questionnaires.getIndexes());

print("\nUser Questionnaire Assignments indexes:");
printjson(db.user_questionnaire_assignments.getIndexes());

print("\nUsers Metadata indexes:");
printjson(db.users_metadata.getIndexes());

print("\n===== Index creation completed! =====");

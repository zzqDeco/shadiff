db = db.getSiblingDB("shadiff");

db.users_old.drop();
db.users_new.drop();

db.users_old.insertOne({
  id: 1,
  name: "Ada Lovelace",
  tier: "gold"
});

db.users_new.insertOne({
  id: 1,
  name: "Ada Lovelace",
  tier: "gold"
});

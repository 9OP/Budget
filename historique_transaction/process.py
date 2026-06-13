#!/usr/bin/env python3

import csv
from decimal import Decimal
from datetime import datetime


CATEGORY_MAPPING = {
    "Alimentation": "Groceries",
    "Banque et assurances": "Finance",
    "Juridique et administratif": "Finance",
    "Logement - maison": "Housing",
    "Loisirs et vacances": "Travel",
    "Revenus et rentrees d'argent": "Income",
    "Sante": "Health",
    "Shopping et services": "Lifestyle",
    "Transports": "Transport",
    "A categoriser - rentree d'argent": "Uncategorized",
    "A categoriser - sortie d'argent": "Uncategorized",
}


SUBCATEGORY_MAPPING = {
    "Restaurant": "Restaurants",
    "Hyper/supermarche": "Groceries",
    "Alimentation - autre": "Groceries",
}


def parse_amount(value: str) -> Decimal:
    if not value:
        return Decimal("0")

    return Decimal(value.replace(",", "."))


def convert_date(value: str) -> str:
    return datetime.strptime(value, "%d/%m/%Y").strftime("%Y-%m-%d")


with open("transactions.csv", encoding="utf-8") as src, \
     open("items.csv", "w", newline="", encoding="utf-8") as dst:

    reader = csv.DictReader(src, delimiter=";")

    writer = csv.DictWriter(
        dst,
        fieldnames=[
            "type",
            "name",
            "amount",
            "date",
            "category",
        ],
    )

    items = []

    for row in reader:
        debit = parse_amount(row["Debit"])
        credit = parse_amount(row["Credit"])

        amount = debit if debit != 0 else credit

        if amount == 0:
            continue

        item_type = "EXPENSE" if amount < 0 else "INCOME"

        category = SUBCATEGORY_MAPPING.get(
            row["Sous categorie"].strip(),
            CATEGORY_MAPPING.get(
                row["Categorie"].strip(),
                "Uncategorized",
            ),
        )

        date = convert_date(row["Date de comptabilisation"])

        items.append({
            "type": item_type,
            "name": row["Libelle simplifie"].strip(),
            "amount": str(abs(amount)),
            "date": date,
            "category": category,
        })

    # latest first
    items.sort(key=lambda item: item["date"], reverse=True)

    for item in items:
        writer.writerow(item)

print("Generated items.csv")

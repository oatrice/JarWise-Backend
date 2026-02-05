#!/usr/bin/env python3
"""
Script to generate sample .mmbak test files for MmbakParser tests.
These are SQLite databases mimicking Money Manager's schema.
"""
import sqlite3
import os

TESTDATA_DIR = "/Users/oatrice/Software-projects/JarWise/Backend/internal/parser/testdata"

def create_schema(conn):
    """Create Money Manager-like schema."""
    cursor = conn.cursor()
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS ASSETS (
            uid TEXT PRIMARY KEY,
            NIC_NAME TEXT,
            TYPE INTEGER
        )
    """)
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS ZCATEGORY (
            uid TEXT PRIMARY KEY,
            NAME TEXT,
            TYPE INTEGER
        )
    """)
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS INOUTCOME (
            uid TEXT PRIMARY KEY,
            ZDATE TEXT,
            ZMONEY REAL,
            DO_TYPE TEXT,
            ZCONTENT TEXT,
            categoryUid TEXT,
            assetUid TEXT
        )
    """)
    conn.commit()

def create_valid_mmbak():
    """Create a valid .mmbak file with correct data."""
    path = os.path.join(TESTDATA_DIR, "valid.mmbak")
    conn = sqlite3.connect(path)
    create_schema(conn)
    cursor = conn.cursor()
    
    # Insert accounts
    cursor.execute("INSERT INTO ASSETS VALUES ('acc1', 'Cash Wallet', 1)")
    cursor.execute("INSERT INTO ASSETS VALUES ('acc2', 'Bank Account', 2)")
    
    # Insert categories
    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat1', 'Food', 0)")
    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat2', 'Salary', 1)")
    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat3', 'Transport', 0)")
    
    # Insert transactions with valid dates
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx1', '2025-01-15', 100.50, '0', 'Lunch', 'cat1', 'acc1')")
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx2', '2025-01-20', 50000.00, '1', 'Monthly Salary', 'cat2', 'acc2')")
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx3', '2025-01-22', 35.00, '0', 'Bus fare', 'cat3', 'acc1')")
    
    conn.commit()
    conn.close()
    print(f"Created: {path}")

def create_bad_dates_mmbak():
    """Create a .mmbak file with various bad date formats."""
    path = os.path.join(TESTDATA_DIR, "bad_dates.mmbak")
    conn = sqlite3.connect(path)
    create_schema(conn)
    cursor = conn.cursor()
    
    # Insert accounts
    cursor.execute("INSERT INTO ASSETS VALUES ('acc1', 'Wallet', 1)")
    
    # Insert categories
    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat1', 'Food', 0)")
    
    # Insert transactions with BAD dates
    # 1. NULL date
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_null_date', NULL, 100.00, '0', 'Null date tx', 'cat1', 'acc1')")
    
    # 2. Empty string date
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_empty_date', '', 200.00, '0', 'Empty date tx', 'cat1', 'acc1')")
    
    # 3. Invalid date string
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_invalid_str', 'not-a-date', 300.00, '0', 'Invalid string', 'cat1', 'acc1')")
    
    # 4. Invalid date format (DD/MM/YYYY instead of YYYY-MM-DD)
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_wrong_format', '32/13/2025', 400.00, '0', 'Wrong format', 'cat1', 'acc1')")
    
    # 5. Partial date
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_partial', '2025-01', 500.00, '0', 'Partial date', 'cat1', 'acc1')")
    
    # 6. Unix timestamp (epoch seconds)
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_unix', '1706745600', 600.00, '0', 'Unix timestamp', 'cat1', 'acc1')")
    
    # 7. Negative timestamp
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_negative', '-12345', 700.00, '0', 'Negative number', 'cat1', 'acc1')")
    
    # 8. Date with time (might be okay, but let's test)
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_datetime', '2025-01-15 14:30:00', 800.00, '0', 'With time', 'cat1', 'acc1')")
    
    # 9. Very old date
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_old', '1900-01-01', 900.00, '0', 'Very old date', 'cat1', 'acc1')")
    
    # 10. Future date (year 9999)
    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_future', '9999-12-31', 1000.00, '0', 'Far future', 'cat1', 'acc1')")
    
    conn.commit()
    conn.close()
    print(f"Created: {path}")

def create_missing_tables_mmbak():
    """Create a .mmbak file with missing required tables."""
    path = os.path.join(TESTDATA_DIR, "missing_tables.mmbak")
    conn = sqlite3.connect(path)
    cursor = conn.cursor()
    
    # Only create ASSETS table, missing ZCATEGORY and INOUTCOME
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS ASSETS (
            uid TEXT PRIMARY KEY,
            NIC_NAME TEXT,
            TYPE INTEGER
        )
    """)
    cursor.execute("INSERT INTO ASSETS VALUES ('acc1', 'Cash', 1)")
    
    conn.commit()
    conn.close()
    print(f"Created: {path}")

def create_empty_mmbak():
    """Create an empty .mmbak file with schema but no records."""
    path = os.path.join(TESTDATA_DIR, "empty.mmbak")
    conn = sqlite3.connect(path)
    create_schema(conn)
    conn.commit()
    conn.close()
    print(f"Created: {path}")

def create_corrupt_mmbak():
    """Create a corrupt file that's not valid SQLite."""
    path = os.path.join(TESTDATA_DIR, "corrupt.mmbak")
    with open(path, 'wb') as f:
        f.write(b"This is not a valid SQLite database file!")
    print(f"Created: {path}")

if __name__ == "__main__":
    os.makedirs(TESTDATA_DIR, exist_ok=True)
    
    create_valid_mmbak()
    create_bad_dates_mmbak()
    create_missing_tables_mmbak()
    create_empty_mmbak()
    create_corrupt_mmbak()
    
    print("\n✅ All test files created successfully!")

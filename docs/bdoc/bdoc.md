# bdoc - on disk document storage engine for flit

bdoc is the read-only document storage engine used by flit. It is designed to handle arbitrary document structures and is optimized for size and fast access.

Variable-length integer encoding is used to minimize the size of the on-disk representation. The database is designed to be read-only, so it can be memory-mapped for fast access.

Document IDs are assigned sequentially starting at 1. It is assumed that every document is in the index, but technically a document ID may be skipped by assigning it 
a length of 0 in the chunk directory. This is used to represent deleted documents.

## On Disk Structure

The bdoc database consists of one or more files on disk, depending on the size of the database and the configured
chunk size.

Each chunk starts with the chunk header and directory, followed by the chunk data. A chunk will contain one or more whole
documents. Documents that exceed the chunk size will be placed into a separate chunk.

## Metadata

The following metadata is expected to be available when a database is opened for reading. This metadata is stored in a JSON file alongside the database files.

 * `formatUUID` - the UUID of the database format
 * `databaseUUID` - a unique identifier for the database
 * `chunks` - an array of the chunks:
	 * `filename` - the name of the chunk file
	 * `size` - the size of the chunk file in bytes
	 * `numDocs` - the number of documents in the chunk
	 * `firstDocId` - the ID of the first document in the chunk
	 * `headerLength` - the length of the chunk header in bytes
 * `keys` - array of the predefined keys, in order of their key codes (0x81 and up)

### Chunk Header

In binary, the chunk header is structured as follows:

 * Chunk Format UUID - 16 bytes - `359e36ac-7022-11f1-a157-b75ec9dab287`
 * Database UUID - 16 bytes - a unique identifier for the database'
 * Chunk number - vint - the chunk number, starting at 0
 * Last chunk - 1 byte - 1 if this is the last chunk, 0 otherwise
 * First document ID - vint - the ID of the first document in the chunk
 * Number of documents - vint - the number of documents in the chunk
 * Length of chunk directory - vint - the length of the chunk directory in bytes

### Chunk Directory

The chunk directory is a list of lengths of each document in the chunk, encoded as variable-length integers. Each document
length is followed by the length of the document header minus one, as a uint8 (max header length is 256 bytes). The document header contains the keys, types, and lengths of each field in the document. The document header is followed by the values of each field in the order they were specified in the header.

## Predefined keys

Up to 126 predefined keys may be specified when the database is created. Predefined keys are stored in the database metadata (JSON) outside the chunk data.
Predefined keys replace string keys in documents and objects with a single byte key code, which reduces the size of the on-disk representation. Predefined keys are assigned key codes starting at 0x81.

Key value 0x80 indicates the end-of-object marker.

Key values less than 0x80 indicate a string key. The remainder of the byte is the beginning of a variable-length integer that encodes the length of the string key in bytes.

## Types

The binary format uses a 3 bit type code to identify the type of each value.

| Type Code | Type Name | Description |
|-----------|-----------|-------------|
| 0x01      | null      | null value |
| 0x02      | bool	 | boolean value |
| 0x03      | int       | integer (up to 64 bits) |
| 0x04      | float     | floating point number |
| 0x05      | string    | UTF-8 encoded string |
| 0x06      | object    | a map of string keys to values |
| 0x07      | array     | an array of values |

### Length Encoding

The type code is the bottom 3 bits of the type byte. The highest bit indicates that the type declaration is followed by a variable-length integer that encodes the length of the value in bytes.
If the high bit is set, the next 2 bits indicate the number of additional bytes to read for the length of the value (1, 2, or 3). 

### Nulls

A null value is represented by a single byte with the value 0x01. Nulls have no length.

In Go, a null is deserialized as the zero value of the type being deserialized. For example, a null string is deserialized as an empty string, and a null int is deserialized as 0.
Only nil pointers are serialized as nulls. Non-nil pointers are serialized as the value they point to, even if that value is the zero value of the type.

In JavaScript, a null is deserialized as the JavaScript null value, and nulls are serialized as the JavaScript null value.

### Booleans

A boolean is represented by the type code 0x04. The value of the boolean is encoded into the type byte itself:

 - `00000010` - false
 - `00001010` - true


### Integers

An integer is represented by the type code 0x03.

Encoding: `0SLLL011`. 

S is the sign bit, where 0 indicates a positive integer and 1 indicates a negative integer. `LLL` indicates the length of the integer in bytes:

0 - 0 bytes (0)
1 - 1 byte (8 bits)
2 - 2 bytes (16 bits)
3 - 3 bytes (24 bits)
4 - 4 bytes (32 bits)
5 - 6 bytes (48 bits)
6 - 8 bytes (64 bits)
7 - reserved



### Floats

A float is represented by the type code 0x04. Floats may be 16, 32, or 64 bits in size. The float header is a variable-length integer holding the type code and the length of the float in bytes.

Encoding: `0LLL0100`, where `LLL` indicates the length of the float in bytes: 0, 2, 4, or 8. Floats of length 0 are treated as 0.0.


### Strings

A string is represented by the type code 0x05. A string header is a variable-length integer holding the type code and the length of the string in bytes.

Strings use Length-Encoding to encode the length of the string in bytes. The maximum string length is 2^24-1 (about 16MB). Strings longer than this are not supported.

Encoding: `1BBLL101` or `0LLLL101`, where `LLLL` is the length of the string in bytes. 

### Objects

An object type is represented by the type code 0x06. An object is a map of string keys to values.  Objects use length encoding to encode the length of the object in bytes. The maximum object length is 2^24-1 (about 16MB). Objects longer than this are not supported.

Encoding: `1BBLL110` or `0LLLL110`, where `LLLL` is the length of the object in bytes. 

Each object starts with a header which holds the keys, types, and lengths of each field in the object. The header is followed by the values of each field in the order they were specified in the header.

For example, a header for the predefined "name" key and a 10 character string value would be:

 - key byte 0x81 (predefined key "name")
 - `01010101` (type code 0x05 for string, length 10)

 The value in the body of the object would be the 10 bytes of the string value.

 However, if the key was a string key "name" instead of a predefined key, the header would be:

 - `00000100` (key of length 4 for string key "name")
 - `01010101` (type code 0x05 for string, length 10)

The format of a key byte is `PXLLLLLL`. If P is set, the key is predefined and the key code is the remainder of the byte. If P is not set, the key is a string. If X is set, the length continues in the next byte. The remainder of the byte is the length of the string key in bytes. The string key is stored in the header immediately following the key byte(s). The maximum length of a string key is 2^14-1 (about 16KB). String keys longer than this are not supported.
In the body of the object, the value would be the 4 bytes of the string key "name" followed by the 10 bytes of the string value.

The header 0x80 indicates the end of the object. The end-of-object marker is not followed by any values.

#### Document limits

The document header may not exceed 255 bytes, nested objects have no such limit. No document field (and thus no nested
field) may exceed 2^24-1 (about 16MB) in length.

Since the last byte of header is the end-of-object marker, the maximum number of fields in a document is 254, depending
on the length of the fields. Max-size fields (16MB) will reduce this limit to 84 fields (with 1 256KB field left over).
This equates to a maximmum document size of just over 1.4GB, but real documents should never come close to this.

### Arrays

An array type is represented by the type code 0x07. An array is a list of values. The array header is a variable-length integer holding the type declaration
for each element, in order. The end of the array header is indicated by the end-of-object marker 0x80. The array header is followed by the values of each element in the order they were specified in the header.
Encoding: `1BBLL111` or `0LLLL111`, where `LLLL` is the length of the array in bytes.

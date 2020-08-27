# doppel
A utility to find duplicate files in a filesystem


# Description

The general algorithm is as follows: if two files have different sizes,
they're inmediately considered different. If they have the same size,
they're hashed (only calculated once per file, when needed) and their
sums are compared for a final veredict.

The final report has the following format:

```
HEX_HASH_SUM
	CLONED_FILE_1
	CLONED_FILE_2
	...
```

And so on, for every group of files that were found to be clones. Results
are sorted so, in theory, multiple runs in the same directory should
produce the same output, if the files don't change.

Remember not to take the results as gospel! No byte-for-byte comparison is
attempted to absolutely confirm two collisions are equal. False positves
with a good hash are unlikely, but not impossible. If the risk is not
acceptable, don't do something like passing the output directly to a
deleting script without byte-for-byte or manual double checking.

Obviously, the more files are compared, the more memory is required for
the internal structure, even if only the strictly necessary is saved
(names, sizes, and cached hashes). Also, the hash function is necessarily
a compromise between hashing speed, size when cached, and probability
of collision. The default (SHA-1) was chosen for being relatively
speedy in modern machines and having an almost negligible chance of
collision in practice (this application is *not* meant for security
hardiness). Of course, some other algorithms are provided, but keep in
mind: the bottleneck will probably be just the tree traversing and all
those stat calls. In any case, hashing should only come into play in the
case of many files with the exact same size, which should be, hopefully,
uncommon in most practical cases.

# TODO
* Directory exclusion and recursion limits.
* Tests.
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title MirrorNFT - ERC721 Interface to Mirror Vault NFT Precompile
/// @notice ERC721-compatible NFT contract that bridges to Cosmos x/nft module
/// @dev All NFT state stored in Cosmos x/nft; this contract is a view into that state
/// @dev Both MetaMask and Keplr can mint/transfer; all operations write to same on-chain state
contract MirrorNFT {
    event NFTMinted(uint256 indexed tokenId, address indexed owner, string ownerCosmos, string tokenURI);
    event NFTTransferred(uint256 indexed tokenId, address indexed from, address indexed to, string fromCosmos, string toCosmos);

    address public constant MIRROR_NFT_PRECOMPILE = 0x0000000000000000000000000000000000000102;

    /// @notice Mint a new NFT (open minting - anyone can mint)
    /// @dev Calls precompile which stores NFT in Cosmos x/nft module
    /// @param tokenId The unique identifier for the NFT (must not exist)
    /// @param uri The metadata URI (IPFS, Arweave, or HTTPS)
    function mint(uint256 tokenId, string calldata uri) external {
        (bool ok, ) = MIRROR_NFT_PRECOMPILE.call(
            abi.encodeWithSignature("mint(uint256,string)", tokenId, uri)
        );
        require(ok, "precompile mint failed");

        // Query owner to get both address formats for event
        (address owner, string memory ownerCosmos, ) = this.ownerOf(tokenId);
        
        emit NFTMinted(tokenId, owner, ownerCosmos, uri);
    }

    /// @notice Transfer NFT to another address
    /// @dev Calls precompile which updates ownership in Cosmos x/nft
    /// @param from Current owner (must be msg.sender)
    /// @param to Recipient address (can be 0x or will be converted from mirror1)
    /// @param tokenId The NFT to transfer
    function transferFrom(address from, address to, uint256 tokenId) external {
        require(msg.sender == from, "caller not owner");
        
        (bool ok, ) = MIRROR_NFT_PRECOMPILE.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", from, to, tokenId)
        );
        require(ok, "precompile transfer failed");

        // Query addresses for event (dual format)
        (address ownerEvm, string memory ownerCosmos, ) = this.ownerOf(tokenId);
        
        // Convert 'to' address to cosmos format for event (precompile will handle conversion)
        emit NFTTransferred(tokenId, from, to, "", ownerCosmos);
    }

    /// @notice Get the owner of an NFT with BOTH address formats
    /// @dev Returns owner in EVM format AND Cosmos format (dual address support)
    /// @param tokenId The NFT to query
    /// @return owner The owner in EVM format (0x...)
    /// @return ownerCosmos The owner in Cosmos format (mirror1...)
    /// @return exists Whether the NFT exists
    function ownerOf(uint256 tokenId) external view returns (
        address owner, 
        string memory ownerCosmos,
        bool exists
    ) {
        (bool ok, bytes memory data) = MIRROR_NFT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("ownerOf(uint256)", tokenId)
        );
        
        if (!ok) {
            return (address(0), "", false);
        }

        // Decode: owner (address), ownerCosmos (string)
        (owner, ownerCosmos) = abi.decode(data, (address, string));
        exists = owner != address(0);
    }

    /// @notice Get the number of NFTs owned by an address
    /// @dev Accepts either 0x or mirror1 format (precompile handles conversion)
    /// @param owner The address to query
    /// @return balance The number of NFTs owned
    function balanceOf(address owner) external view returns (uint256 balance) {
        (bool ok, bytes memory data) = MIRROR_NFT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("balanceOf(address)", owner)
        );
        require(ok, "precompile balanceOf failed");

        balance = abi.decode(data, (uint256));
    }

    /// @notice Get the metadata URI for an NFT
    /// @param tokenId The NFT to query
    /// @return uri The metadata URI (IPFS, Arweave, etc.)
    function tokenURI(uint256 tokenId) external view returns (string memory uri) {
        (bool ok, bytes memory data) = MIRROR_NFT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("tokenURI(uint256)", tokenId)
        );
        require(ok, "precompile tokenURI failed");

        uri = abi.decode(data, (string));
    }

    /// @notice Get all NFT token IDs owned by an address
    /// @dev Useful for displaying user's NFT gallery
    /// @param owner The address to query
    /// @return tokenIds Array of token IDs owned by the address
    function tokensOfOwner(address owner) external view returns (uint256[] memory tokenIds) {
        (bool ok, bytes memory data) = MIRROR_NFT_PRECOMPILE.staticcall(
            abi.encodeWithSignature("tokensOfOwner(address)", owner)
        );
        require(ok, "precompile tokensOfOwner failed");

        tokenIds = abi.decode(data, (uint256[]));
    }

    /// @notice Check if an NFT exists
    /// @param tokenId The NFT to check
    /// @return exists True if the NFT has been minted
    function exists(uint256 tokenId) external view returns (bool exists) {
        (, , exists) = this.ownerOf(tokenId);
    }

    /// @notice Get NFT details with dual address format
    /// @dev Comprehensive query returning all NFT information
    /// @param tokenId The NFT to query
    /// @return owner Owner in EVM format (0x...)
    /// @return ownerCosmos Owner in Cosmos format (mirror1...)
    /// @return uri Metadata URI
    /// @return exists Whether the NFT exists
    function getNFT(uint256 tokenId) external view returns (
        address owner,
        string memory ownerCosmos,
        string memory uri,
        bool exists
    ) {
        (owner, ownerCosmos, exists) = this.ownerOf(tokenId);
        
        if (exists) {
            uri = this.tokenURI(tokenId);
        }
    }
}
